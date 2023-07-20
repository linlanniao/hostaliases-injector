package v1

import (
	"context"
	"fmt"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var joblog = logf.Log.WithName("job-mutator")

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-batch-v1-job,mutating=true,failurePolicy=ignore,sideEffects=None,groups=batch,resources=jobs,verbs=create;update,versions=v1,name=mjob.kb.io,admissionReviewVersions=v1
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete

//var _ webhook.Defaulter = &Job{}
//
//// Default implements webhook.Defaulter so a webhook will be registered for the type
//func (r *Job) Default() {
//	joblog.Info("default", "name", r.Name)
//
//	// TODO(user): fill in your defaulting logic.
//}

type JobMutate struct {
	Client  client.Client
	decoder *admission.Decoder
}

var _ admission.Handler = &JobMutate{}

func NewJobMutate(client client.Client) admission.Handler {
	decoder := admission.NewDecoder(runtime.NewScheme())
	return &JobMutate{
		Client:  client,
		decoder: decoder,
	}
}

func (v *JobMutate) Handle(ctx context.Context, req admission.Request) admission.Response {
	// TODO
	job := &batchv1.Job{}
	joblog.Info(fmt.Sprintf("%#v", v))
	joblog.Info(fmt.Sprintf("%#v", v.decoder))
	joblog.Info(fmt.Sprintf("%#v", req))
	//joblog.Info(fmt.Sprintf("%#v", v.decoder.Decode(req, nil)))

	err := v.decoder.Decode(req, job)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	//job.Labels["testaaaaa"] = "testbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

	resp, err := json.Marshal(job)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	joblog.Info("labeled", job.GetLabels())
	return admission.PatchResponseFromRaw(req.Object.Raw, resp)
}

//func (v *JobMutate) InjectDecoder(d *admission.Decoder) error {
//	v.decoder = d
//	return nil
//}
