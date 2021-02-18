/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/MOZGIII/port-map-operator/pkg/annotations"
	"github.com/MOZGIII/port-map-operator/pkg/portmap"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ServiceReconciler reconciles a Service object
type ServiceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	PortMap         portmap.Mapper
	DefaultLifetime portmap.Lifetime
}

//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=services/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("service", req.NamespacedName)

	var service corev1.Service
	if err := r.Get(ctx, req.NamespacedName, &service); err != nil {
		log.Error(err, "unable to fetch Service, skipping")
		// We'll ignore not-found errors, since they can't be fixed by
		// an immediate requeue (we'll need to wait for a new notification),
		// and we can get them on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if service.Spec.Type != corev1.ServiceTypeLoadBalancer {
		log.V(1).Info("service type is not LoadBalancer, skipping")
		return ctrl.Result{}, nil
	}

	ann, err := annotations.FromService(&service)
	if err != nil {
		log.V(1).Info("no annotations found, skipping")
		return ctrl.Result{}, nil
	}

	pmreqlist, err := makePortmapRequests(log, &service, ann, r.DefaultLifetime)
	if err != nil {
		log.V(1).Info("unable to produce portmap request, skipping")
		return ctrl.Result{}, nil
	}

	pmreslist, pmerrlist := r.mapPorts(ctx, log, pmreqlist)
	log.Info("port mapping procedute finished", "errors", pmerrlist, "responses", pmreslist)

	err = r.updateStatus(ctx, &service, pmreslist, pmerrlist)
	if err != nil {
		log.Error(err, "unable to update the Service status")
	} else {
		log.V(2).Info("status updated successfully")
	}

	requeueAfter := r.DefaultLifetime - 2
	const twelveHoursInSecs = 12 * 60
	if requeueAfter > twelveHoursInSecs {
		requeueAfter = twelveHoursInSecs
	}

	return ctrl.Result{
		// Force reqeue this service to renew the port map lifetime.
		RequeueAfter: time.Second * time.Duration(requeueAfter),
	}, client.IgnoreNotFound(err)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Service{}).
		Complete(r)
}

func makePortmapRequests(log logr.Logger, service *corev1.Service, ann *annotations.Annotations, defaultLifetime portmap.Lifetime) ([]*portmap.Request, error) {
	pmreqlist := make([]*portmap.Request, 0, len(service.Spec.Ports))

	for _, servicePort := range service.Spec.Ports {
		var protocol portmap.Protocol
		switch servicePort.Protocol {
		case corev1.ProtocolTCP:
			protocol = portmap.ProtocolTCP
		case corev1.ProtocolUDP:
			protocol = portmap.ProtocolUDP
		case corev1.ProtocolSCTP:
			protocol = portmap.ProtocolSCTP
		default:
			log.Info("unexpected protocol", "protocol", servicePort.Protocol)
		}

		pmreq := &portmap.Request{
			Protocol:    protocol,
			NodePort:    portmap.Port(servicePort.NodePort),
			GatewayPort: portmap.Port(servicePort.Port),
			Lifetime:    defaultLifetime,
		}
		pmreqlist = append(pmreqlist, pmreq)
	}

	return pmreqlist, nil
}

func (r *ServiceReconciler) mapPorts(ctx context.Context, log logr.Logger, pmreqlist []*portmap.Request) ([]*portmap.Response, []error) {
	log.V(1).Info("mapping ports", "requests", pmreqlist)

	var (
		pmreslist []*portmap.Response
		pmerrlist []error
	)
	for _, pmreq := range pmreqlist {
		pmres, err := r.mapPort(ctx, log, pmreq)
		if err != nil {
			pmerrlist = append(pmerrlist, err)
			continue
		}
		pmreslist = append(pmreslist, pmres)
	}
	return pmreslist, pmerrlist
}

type ErrMappedGatewayPortMismatch struct {
	RequestedGatewayPort portmap.Port
	MappedGatewayPort    portmap.Port
}

var _ error = (*ErrMappedGatewayPortMismatch)(nil)

func (e *ErrMappedGatewayPortMismatch) Error() string {
	return fmt.Sprintf(
		"mapped gateway port (%d) is different from the requested port (%d)",
		e.MappedGatewayPort, e.RequestedGatewayPort,
	)
}

func (r *ServiceReconciler) mapPort(ctx context.Context, log logr.Logger, pmreq *portmap.Request) (*portmap.Response, error) {
	log.V(1).Info("mapping port", "request", pmreq)

	pmres, err := r.PortMap.Map(ctx, pmreq)
	if err != nil {
		log.Error(err, "unable to map the port", "request", pmreq)
		return nil, err
	}

	if err := checkRequestResponseCoherence(pmreq, pmres); err != nil {
		log.Error(err, "the response was not coherent to the request", "request", pmreq, "response", pmres)

		cancelreq := &portmap.Request{
			Protocol:    pmres.Protocol,
			NodePort:    pmres.NodePort,
			GatewayPort: pmres.GatewayPort,
			Lifetime:    portmap.LifetimeDelete,
		}
		cancelres, cancelerr := r.PortMap.Map(ctx, cancelreq)
		if cancelerr != nil {
			log.Error(
				cancelerr,
				"failed to cancel incoherent port map",
				"request", pmreq, "response", pmres,
				"cancelreq", cancelreq, "cancelres", cancelres,
			)
		}

		return nil, err
	}

	return pmres, nil
}

func checkRequestResponseCoherence(pmreq *portmap.Request, pmres *portmap.Response) error {
	if pmres.GatewayPort != pmreq.GatewayPort {
		return &ErrMappedGatewayPortMismatch{
			RequestedGatewayPort: pmreq.GatewayPort,
			MappedGatewayPort:    pmres.GatewayPort,
		}
	}
	return nil
}

func (r *ServiceReconciler) updateStatus(ctx context.Context, service *corev1.Service, pmreslist []*portmap.Response, pmerrlist []error) error {
	serviceCopy := service.DeepCopy()

	extenralIPs := make([]string, 0, len(pmreslist))

OuterLoop:
	for _, pmres := range pmreslist {
		ip := pmres.GatewayIP.String()
		for _, existingIP := range extenralIPs {
			if ip == existingIP {
				continue OuterLoop
			}
		}
		extenralIPs = append(extenralIPs, ip)
	}

	serviceCopy.Spec.ExternalIPs = extenralIPs

	// TODO: set load balancer ingresses
	// TODO: set proper conditions

	return r.Update(ctx, serviceCopy)
}
