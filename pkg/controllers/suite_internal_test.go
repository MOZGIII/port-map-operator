package controllers

import (
	"math/rand"
	"path/filepath"
	"testing"
	"time"

	"github.com/MOZGIII/port-map-operator/pkg/pmmock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	//+kubebuilder:scaffold:imports
)

var cfg *rest.Config
var k8sClient client.Client
var k8sManager ctrl.Manager
var testEnv *envtest.Environment
var portMapper *pmmock.MockMapper
var pmmockctl *pmmock.Control

const defaultLifetime = 120

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: false,
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = corev1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	portMapper, pmmockctl = pmmock.New()

	k8sManager, err = ctrl.NewManager(cfg, ctrl.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sManager).NotTo(BeNil())

	k8sClient = k8sManager.GetClient()
	Expect(k8sClient).ToNot(BeNil())

	err = (&ServiceReconciler{
		Client: k8sClient,
		Log:    ctrl.Log.WithName("controllers").WithName("Service"),
		Scheme: k8sManager.GetScheme(),

		PortMap:         portMapper,
		DefaultLifetime: defaultLifetime,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		err = k8sManager.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred())
	}()

	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz1234567890")

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		// nolint: gosec
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
