package v1

import (
	"context"
	"flag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"testing"
)

func newClient() client.Client {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}

	c, err := client.New(config, client.Options{})
	if err != nil {
		panic(err)
	}

	return c

}

func TestGetJob(t *testing.T) {
	c := newClient()

	jm := JobMutate{
		Client:  c,
		decoder: admission.NewDecoder(runtime.NewScheme()),
	}
	job, err := jm.GetJob(context.TODO(), "a", "")
	if err != nil {
		t.Error(err.Error())
	}
	t.Log(job.String())

}
