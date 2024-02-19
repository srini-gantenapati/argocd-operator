package argocd

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/google/go-cmp/cmp"
	configv1 "github.com/openshift/api/config/v1"
	routev1 "github.com/openshift/api/route/v1"

	argoproj "github.com/argoproj-labs/argocd-operator/api/v1beta1"
)

func TestReconcileRouteSetLabels(t *testing.T) {
	routeAPIFound = true

	ctx := context.Background()
	logf.SetLogger(ZapLogger(true))
	argoCD := makeArgoCD(func(a *argoproj.ArgoCD) {
		a.Spec.Server.Route.Enabled = true
		labels := make(map[string]string)
		labels["my-key"] = "my-value"
		a.Spec.Server.Route.Labels = labels
	})

	resObjs := []client.Object{argoCD}
	subresObjs := []client.Object{argoCD}
	runtimeObjs := []runtime.Object{}
	sch := makeTestReconcilerScheme(argoproj.AddToScheme, configv1.Install, routev1.Install)
	cl := makeTestReconcilerClient(sch, resObjs, subresObjs, runtimeObjs)
	r := makeTestReconciler(cl, sch)

	assert.NoError(t, createNamespace(r, argoCD.Namespace, ""))

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      testArgoCDName,
			Namespace: testNamespace,
		},
	}

	_, err := r.Reconcile(context.TODO(), req)
	assert.NoError(t, err)

	loaded := &routev1.Route{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: testArgoCDName + "-server", Namespace: testNamespace}, loaded)
	fatalIfError(t, err, "failed to load route %q: %s", testArgoCDName+"-server", err)

	if diff := cmp.Diff("my-value", loaded.Labels["my-key"]); diff != "" {
		t.Fatalf("failed to reconcile route:\n%s", diff)
	}

}
func TestReconcileRouteSetsInsecure(t *testing.T) {
	routeAPIFound = true

	ctx := context.Background()
	logf.SetLogger(ZapLogger(true))
	argoCD := makeArgoCD(func(a *argoproj.ArgoCD) {
		a.Spec.Server.Route.Enabled = true
	})

	resObjs := []client.Object{argoCD}
	subresObjs := []client.Object{argoCD}
	runtimeObjs := []runtime.Object{}
	sch := makeTestReconcilerScheme(argoproj.AddToScheme, configv1.Install, routev1.Install)
	cl := makeTestReconcilerClient(sch, resObjs, subresObjs, runtimeObjs)
	r := makeTestReconciler(cl, sch)

	assert.NoError(t, createNamespace(r, argoCD.Namespace, ""))

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      testArgoCDName,
			Namespace: testNamespace,
		},
	}

	_, err := r.Reconcile(context.TODO(), req)
	assert.NoError(t, err)

	loaded := &routev1.Route{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: testArgoCDName + "-server", Namespace: testNamespace}, loaded)
	fatalIfError(t, err, "failed to load route %q: %s", testArgoCDName+"-server", err)

	wantTLSConfig := &routev1.TLSConfig{
		Termination:                   routev1.TLSTerminationPassthrough,
		InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyRedirect,
	}
	if diff := cmp.Diff(wantTLSConfig, loaded.Spec.TLS); diff != "" {
		t.Fatalf("failed to reconcile route:\n%s", diff)
	}
	wantPort := &routev1.RoutePort{
		TargetPort: intstr.FromString("https"),
	}
	if diff := cmp.Diff(wantPort, loaded.Spec.Port); diff != "" {
		t.Fatalf("failed to reconcile route:\n%s", diff)
	}

	// second reconciliation after changing the Insecure flag.
	err = r.Client.Get(ctx, req.NamespacedName, argoCD)
	fatalIfError(t, err, "failed to load ArgoCD %q: %s", testArgoCDName+"-server", err)

	argoCD.Spec.Server.Insecure = true
	err = r.Client.Update(ctx, argoCD)
	fatalIfError(t, err, "failed to update the ArgoCD: %s", err)

	_, err = r.Reconcile(context.TODO(), req)
	fatalIfError(t, err, "reconcile: (%v): %s", req, err)

	loaded = &routev1.Route{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: testArgoCDName + "-server", Namespace: testNamespace}, loaded)
	fatalIfError(t, err, "failed to load route %q: %s", testArgoCDName+"-server", err)

	wantTLSConfig = &routev1.TLSConfig{
		InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyRedirect,
		Termination:                   routev1.TLSTerminationEdge,
	}
	if diff := cmp.Diff(wantTLSConfig, loaded.Spec.TLS); diff != "" {
		t.Fatalf("failed to reconcile route:\n%s", diff)
	}
	wantPort = &routev1.RoutePort{
		TargetPort: intstr.FromString("http"),
	}
	if diff := cmp.Diff(wantPort, loaded.Spec.Port); diff != "" {
		t.Fatalf("failed to reconcile route:\n%s", diff)
	}
}

func TestReconcileRouteUnsetsInsecure(t *testing.T) {
	routeAPIFound = true
	ctx := context.Background()
	logf.SetLogger(ZapLogger(true))
	argoCD := makeArgoCD(func(a *argoproj.ArgoCD) {
		a.Spec.Server.Route.Enabled = true
		a.Spec.Server.Insecure = true
	})

	resObjs := []client.Object{argoCD}
	subresObjs := []client.Object{argoCD}
	runtimeObjs := []runtime.Object{}
	sch := makeTestReconcilerScheme(argoproj.AddToScheme, configv1.Install, routev1.Install)
	cl := makeTestReconcilerClient(sch, resObjs, subresObjs, runtimeObjs)
	r := makeTestReconciler(cl, sch)

	assert.NoError(t, createNamespace(r, argoCD.Namespace, ""))

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      testArgoCDName,
			Namespace: testNamespace,
		},
	}

	_, err := r.Reconcile(context.TODO(), req)
	assert.NoError(t, err)

	loaded := &routev1.Route{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: testArgoCDName + "-server", Namespace: testNamespace}, loaded)
	fatalIfError(t, err, "failed to load route %q: %s", testArgoCDName+"-server", err)

	wantTLSConfig := &routev1.TLSConfig{
		InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyRedirect,
		Termination:                   routev1.TLSTerminationEdge,
	}
	if diff := cmp.Diff(wantTLSConfig, loaded.Spec.TLS); diff != "" {
		t.Fatalf("failed to reconcile route:\n%s", diff)
	}
	wantPort := &routev1.RoutePort{
		TargetPort: intstr.FromString("http"),
	}
	if diff := cmp.Diff(wantPort, loaded.Spec.Port); diff != "" {
		t.Fatalf("failed to reconcile route:\n%s", diff)
	}

	// second reconciliation after changing the Insecure flag.
	err = r.Client.Get(ctx, req.NamespacedName, argoCD)
	fatalIfError(t, err, "failed to load ArgoCD %q: %s", testArgoCDName+"-server", err)

	argoCD.Spec.Server.Insecure = false
	err = r.Client.Update(ctx, argoCD)
	fatalIfError(t, err, "failed to update the ArgoCD: %s", err)

	_, err = r.Reconcile(context.TODO(), req)
	assert.NoError(t, err)

	loaded = &routev1.Route{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: testArgoCDName + "-server", Namespace: testNamespace}, loaded)
	fatalIfError(t, err, "failed to load route %q: %s", testArgoCDName+"-server", err)

	wantTLSConfig = &routev1.TLSConfig{
		Termination:                   routev1.TLSTerminationPassthrough,
		InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyRedirect,
	}
	if diff := cmp.Diff(wantTLSConfig, loaded.Spec.TLS); diff != "" {
		t.Fatalf("failed to reconcile route:\n%s", diff)
	}
	wantPort = &routev1.RoutePort{
		TargetPort: intstr.FromString("https"),
	}
	if diff := cmp.Diff(wantPort, loaded.Spec.Port); diff != "" {
		t.Fatalf("failed to reconcile route:\n%s", diff)
	}
}

func TestReconcileRouteForShorteningHostname(t *testing.T) {
	routeAPIFound = true
	ctx := context.Background()
	logf.SetLogger(ZapLogger(true))

	tests := []struct {
		testName string
		expected string
		hostname string
	}{
		{
			testName: "longHostname",
			hostname: "myhostnameaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.redhat.com",
			expected: "myhostnameaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.redhat.com",
		},
		{
			testName: "twentySixLetterHostname",
			hostname: "myhostnametwentysixletteraaaaaaaaaaaaaaaaaaaaaa.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.redhat.com",
			expected: "myhostnametwentysixletteraaaaaaaaaaaaaa.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.redhat.com",
		},
	}

	for _, v := range tests {
		t.Run(v.testName, func(t *testing.T) {

			argoCD := makeArgoCD(func(a *argoproj.ArgoCD) {
				a.Spec.Server.Route.Enabled = true
				a.Spec.ApplicationSet = &argoproj.ArgoCDApplicationSet{
					WebhookServer: argoproj.WebhookServerSpec{
						Route: argoproj.ArgoCDRouteSpec{
							Enabled: true,
						},
						Host: v.hostname,
					},
				}
			})

			resObjs := []client.Object{argoCD}
			subresObjs := []client.Object{argoCD}
			runtimeObjs := []runtime.Object{}
			sch := makeTestReconcilerScheme(argoproj.AddToScheme, configv1.Install, routev1.Install)
			cl := makeTestReconcilerClient(sch, resObjs, subresObjs, runtimeObjs)
			r := makeTestReconciler(cl, sch)

			assert.NoError(t, createNamespace(r, argoCD.Namespace, ""))

			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      testArgoCDName,
					Namespace: testNamespace,
				},
			}

			// Check if it returns nil when hostname is empty
			_, err := r.Reconcile(context.TODO(), req)
			assert.NoError(t, err)

			// second reconciliation after changing the hostname.
			err = r.Client.Get(ctx, req.NamespacedName, argoCD)
			fatalIfError(t, err, "failed to load ArgoCD %q: %s", testArgoCDName+"-server", err)

			argoCD.Spec.Server.Host = v.hostname
			err = r.Client.Update(ctx, argoCD)
			fatalIfError(t, err, "failed to update the ArgoCD: %s", err)

			_, err = r.Reconcile(context.TODO(), req)
			assert.NoError(t, err)

			loaded := &routev1.Route{}
			err = r.Client.Get(ctx, types.NamespacedName{Name: testArgoCDName + "-server", Namespace: testNamespace}, loaded)
			fatalIfError(t, err, "failed to load route %q: %s", testArgoCDName+"-server", err)

			if diff := cmp.Diff(v.expected, loaded.Spec.Host); diff != "" {
				t.Fatalf("failed to reconcile route:\n%s", diff)
			}

			// Check if first label is greater than 20
			labels := strings.Split(loaded.Spec.Host, ".")
			assert.True(t, len(labels[0]) > 20)

		})
	}
}

func makeArgoCD(opts ...func(*argoproj.ArgoCD)) *argoproj.ArgoCD {
	argoCD := &argoproj.ArgoCD{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testArgoCDName,
			Namespace: testNamespace,
		},
		Spec: argoproj.ArgoCDSpec{},
	}
	for _, o := range opts {
		o(argoCD)
	}
	return argoCD
}

func fatalIfError(t *testing.T, err error, format string, a ...interface{}) {
	t.Helper()
	if err != nil {
		t.Fatalf(format, a...)
	}
}
