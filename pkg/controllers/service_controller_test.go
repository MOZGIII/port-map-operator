package controllers

import (
	"context"
	"fmt"
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
		serviceName = "test-service"

		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	var (
		serviceNamespace string
		ctx              context.Context
		namespace        *corev1.Namespace
	)

	BeforeEach(func() {
		ctx = context.Background()
		Expect(cfg).NotTo(BeNil())

		By("By creating a new Namespace")
		serviceNamespace = fmt.Sprintf("test-service-ns-%s", randStringRunes(5))
		namespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: serviceNamespace},
		}
		err := k8sClient.Create(ctx, namespace)
		Expect(err).NotTo(HaveOccurred(), "failed to create test namespace")
	})

	AfterEach(func() {
		Eventually(func() error {
			return k8sClient.Delete(context.Background(), namespace)
		}, timeout, interval).Should(Succeed(), "failed to delete test namespace")
	})

	Context("When mapping ports for LoadBalancer Service", func() {
		It("Should issue proper port map requests", func() {
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
							Port:     1234,
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
			Eventually(func() error {
				return k8sClient.Get(ctx, serviceLookupKey, createdService)
			}, timeout, interval).Should(Succeed())
			Expect(createdService.ObjectMeta.Name).Should(Equal(serviceName))

			By("By waiting for the mock port mapper to receive the port map request")
			pmmockctl.Expect(&portmap.Request{
				Protocol:    portmap.ProtocolTCP,
				NodePort:    portmap.Port(32100),
				GatewayPort: portmap.Port(1234),
				Lifetime:    portmap.Lifetime(120),
			}, timeout)
			pmmockctl.Inject(&portmap.Response{
				Protocol:    portmap.ProtocolTCP,
				NodePort:    portmap.Port(32100),
				GatewayPort: portmap.Port(1234),
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
			}, timeout, interval).Should(ConsistOf("1.2.3.4"), "should list the mapped IP in the external IPs")

			By("Deleting the service after the test is done")
			autostopch := pmmockctl.Auto()
			defer close(autostopch)
			Expect(k8sClient.Delete(ctx, service)).Should(Succeed())
		})
	})

	Context("When mapped port does not match the requested port", func() {
		It("Should reject the the mapping", func() {
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
			Eventually(func() error {
				return k8sClient.Get(ctx, serviceLookupKey, createdService)
			}, timeout, interval).Should(Succeed())
			Expect(createdService.ObjectMeta.Name).Should(Equal(serviceName))

			By("By waiting for the mock port mapper to receive the port map request")
			pmmockctl.Expect(&portmap.Request{
				Protocol:    portmap.ProtocolTCP,
				NodePort:    portmap.Port(32100),
				GatewayPort: portmap.Port(256),
				Lifetime:    portmap.Lifetime(120),
			}, timeout)
			By("By injecting a mock port mapper response with non-matching gateway port")
			pmmockctl.Inject(&portmap.Response{
				Protocol:    portmap.ProtocolTCP,
				NodePort:    portmap.Port(32100),
				GatewayPort: portmap.Port(1024),
				GatewayIP:   net.IPv4(1, 2, 3, 4),
				Lifetime:    portmap.Lifetime(120),
			}, timeout)

			By("By waiting for a lease cancel")
			pmmockctl.Expect(&portmap.Request{
				Protocol:    portmap.ProtocolTCP,
				NodePort:    portmap.Port(32100),
				GatewayPort: portmap.Port(1024),
				Lifetime:    portmap.LifetimeDelete,
			}, timeout)
			By("By injecting a mock port mapper response with deletion accept")
			pmmockctl.Inject(&portmap.Response{
				Protocol:    portmap.ProtocolTCP,
				NodePort:    portmap.Port(32100),
				GatewayPort: portmap.Port(1024),
				GatewayIP:   net.IPv4(1, 2, 3, 4),
				Lifetime:    portmap.LifetimeDelete,
			}, timeout)

			By("By checking that the ExternalIPs is not updated")
			Eventually(func() ([]string, error) {
				err := k8sClient.Get(ctx, serviceLookupKey, createdService)
				if err != nil {
					return nil, err
				}

				return createdService.Spec.ExternalIPs, nil
			}, timeout, interval).Should(BeEmpty(), "should be empty")

			By("Deleting the service after the test is done")
			autostopch := pmmockctl.Auto()
			defer close(autostopch)
			Expect(k8sClient.Delete(ctx, service)).Should(Succeed())
		})
	})

	Context("When multiple ports are mapped", func() {
		It("Should only include the IP once", func() {
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
							Port:     1100,
							NodePort: 32100,
						},
						{
							Name:     "test2",
							Protocol: "TCP",
							Port:     1101,
							NodePort: 32101,
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
			Eventually(func() error {
				return k8sClient.Get(ctx, serviceLookupKey, createdService)
			}, timeout, interval).Should(Succeed())
			Expect(createdService.ObjectMeta.Name).Should(Equal(serviceName))

			By("By waiting for the mock port mapper to receive the port map request for test1 port")
			pmmockctl.Expect(&portmap.Request{
				Protocol:    portmap.ProtocolTCP,
				NodePort:    portmap.Port(32100),
				GatewayPort: portmap.Port(1100),
				Lifetime:    portmap.Lifetime(120),
			}, timeout)
			pmmockctl.Inject(&portmap.Response{
				Protocol:    portmap.ProtocolTCP,
				NodePort:    portmap.Port(32100),
				GatewayPort: portmap.Port(1100),
				GatewayIP:   net.IPv4(1, 2, 3, 4),
				Lifetime:    portmap.Lifetime(120),
			}, timeout)

			By("By waiting for the mock port mapper to receive the port map request for test2 port")
			pmmockctl.Expect(&portmap.Request{
				Protocol:    portmap.ProtocolTCP,
				NodePort:    portmap.Port(32101),
				GatewayPort: portmap.Port(1101),
				Lifetime:    portmap.Lifetime(120),
			}, timeout)
			pmmockctl.Inject(&portmap.Response{
				Protocol:    portmap.ProtocolTCP,
				NodePort:    portmap.Port(32101),
				GatewayPort: portmap.Port(1101),
				GatewayIP:   net.IPv4(1, 2, 3, 4),
				Lifetime:    portmap.Lifetime(120),
			}, timeout)

			By("By checking that the ExternalIPs contains just one value")
			Eventually(func() ([]string, error) {
				err := k8sClient.Get(ctx, serviceLookupKey, createdService)
				if err != nil {
					return nil, err
				}

				return createdService.Spec.ExternalIPs, nil
			}, timeout, interval).Should(ConsistOf("1.2.3.4"), "should list the mapped IP in the external IPs once")

			By("Deleting the service after the test is done")
			autostopch := pmmockctl.Auto()
			defer close(autostopch)
			Expect(k8sClient.Delete(ctx, service)).Should(Succeed())
		})
	})
})
