package v1

import (
	"context"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

//+kubebuilder:webhook:path=/mutate-core-v1-pod,mutating=true,failurePolicy=ignore,sideEffects=None,groups="",resources=pods,verbs=create,versions=v1,name=mjob.kb.io,admissionReviewVersions=v1
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list

// log is for logging in this package.
var logger = logf.Log.WithName("hostaliases-injector")

type Mutate struct {
	Client  client.Client
	decoder *admission.Decoder
}

func NewMutate(client client.Client) admission.Handler {
	decoder := admission.NewDecoder(runtime.NewScheme())
	return &Mutate{
		Client:  client,
		decoder: decoder,
	}
}

const (
	watchingLabelKey = "k8s-app"
)

func (m *Mutate) Handle(ctx context.Context, req admission.Request) admission.Response {

	pod := new(corev1.Pod)
	if err := m.decoder.Decode(req, pod); err != nil {
		logger.Error(err, "failed to decode pod")
		return admission.Errored(http.StatusBadRequest, err)
	}

	if _, ok := pod.Labels[watchingLabelKey]; !ok {
		logger.Info(
			"uncontrolled target",
			"action", "skip",
			"namespace", pod.Namespace,
			"name", pod.Name,
		)
		return admission.Allowed("")
	}

	hostAliases, err := m.getServiceHostsAliases()
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	pod.Spec.HostAliases = hostAliases
	logger.Info(
		"injected host aliases",
		"namespace", pod.Namespace,
		"name", pod.Name,
	)

	resp, err := json.Marshal(pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, resp)
}
