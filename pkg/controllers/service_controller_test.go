package controllers

import (
	"context"
	"net"
	"time"

	"github.com/MOZGIII/port-map-operator/pkg/portmap"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Service controller", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		serviceName      = "test-service"
		serviceNamespace = "test-service-namespace"

		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When mapping ports for LoadBalancer Service", func() {
		It("Should issue proper port map requests", func() {
			ctx := context.Background()

			By("creating the test Namespace")
			namespace := &corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					APIVersion: corev1.SchemeGroupVersion.Version,
					Kind:       "Namespace",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: serviceNamespace,
				},
				Spec: corev1.NamespaceSpec{},
			}
			Expect(k8sClient.Create(ctx, namespace)).Should(Succeed())

			By("By creating a new Service")
			service := &corev1.Service{
				TypeMeta: metav1.TypeMeta{
					APIVersion: corev1.SchemeGroupVersion.Version,
					Kind:       "Service",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      serviceName,
					Namespace: serviceNamespace,
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Name:     "test1",
							Protocol: "TCP",
							Port:     256,
							NodePort: 32100,
						},
					},
					Selector: map[string]string{
						"app": "test",
					},
					Type: corev1.ServiceTypeLoadBalancer,
				},
			}
			Expect(k8sClient.Create(ctx, service)).Should(Succeed())
			serviceLookupKey := types.NamespacedName{Name: serviceName, Namespace: serviceNamespace}

			By("By waiting for the created Service to appear at the API")
			createdService := &corev1.Service{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, serviceLookupKey, createdService)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(createdService.ObjectMeta.Name).Should(Equal(serviceName))

			By("By waiting for the mock port mapper to receive the port map request")
			pmockctl.Expect(&portmap.Request{
				Protocol:    portmap.ProtocolTCP,
				NodePort:    portmap.Port(32100),
				GatewayPort: portmap.Port(256),
				Lifetime:    portmap.Lifetime(120),
			}, timeout)
			pmockctl.Inject(&portmap.Response{
				Protocol:    portmap.ProtocolTCP,
				NodePort:    portmap.Port(32100),
				GatewayPort: portmap.Port(256),
				GatewayIP:   net.IPv4(1, 2, 3, 4),
				Lifetime:    portmap.Lifetime(120),
			}, timeout)

			By("By checking that the ExternalIPs is updated")
			Eventually(func() ([]string, error) {
				err := k8sClient.Get(ctx, serviceLookupKey, createdService)
				if err != nil {
					return nil, err
				}

				return createdService.Spec.ExternalIPs, nil
			}, timeout, interval).Should(ConsistOf("1.2.3.4"), "should list the mapped IP in the enternal IPs")
		})
	})

})
