package annotations

import (
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Annotations", func() {
	When("not set", func() {
		It("should be empty", func() {
			service := corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "test-service"}}
			ann, err := FromService(&service)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ann).To(Equal(&Annotations{Overrides: make(Overrides)}))
		})
	})

	When("set to a valid value", func() {
		It("should parse properly", func() {
			service := corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "test-service", Annotations: map[string]string{
				OverridesV1Key: `{"UDP/5000": {"skip": true}, "TCP/3000": {"port": 80}}`,
			}}}
			ann, err := FromService(&service)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ann).To(Equal(&Annotations{Overrides: Overrides{
				PortDescriptor{
					Protocol: "UDP",
					Port:     5000,
				}: &Override{
					Skip: true,
				},
				PortDescriptor{
					Protocol: "TCP",
					Port:     3000,
				}: &Override{
					Port: 80,
				},
			}}))
		})
	})

	When("set to an invalid value", func() {
		It("should return a JSON parsing error", func() {
			service := corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "test-service", Annotations: map[string]string{
				OverridesV1Key: `{`,
			}}}
			ann, err := FromService(&service)
			Expect(ann).To(BeNil())
			Expect(err).ShouldNot(MatchError(&json.SyntaxError{}))
		})
	})
})
