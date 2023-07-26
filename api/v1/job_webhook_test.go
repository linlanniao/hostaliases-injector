package v1

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/admission/v1"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"testing"
)

func newJm() JobMutate {
	cfg, err := config.GetConfig()
	c, err := client.New(cfg, client.Options{})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return JobMutate{
		Client:  c,
		decoder: admission.NewDecoder(runtime.NewScheme()),
	}
}

func TestGetJob(t *testing.T) {
	jm := newJm()
	job, err := jm.GetJob(context.TODO(), "test-job-latest", "")
	if err != nil {
		t.Log(err.Error())
	}
	t.Log(job.String())
}

func TestDeleteJob(t *testing.T) {
	jm := newJm()
	job, err := jm.GetJob(context.TODO(), "test-job-latest", "")
	if err != nil {
		t.Log(err.Error())
	}
	t.Log(job.String())
	t.Log("--------------------------------")
	err2 := jm.DeleteJob(context.TODO(), job.Name, job.Namespace)
	if err2 != nil {
		t.Log(err.Error())
	}
}

func TestReplaceJob(t *testing.T) {
	jm := newJm()
	job, err := jm.GetJob(context.TODO(), "test-job-latest", "")
	if err != nil {
		t.Log(err.Error())
	}
	t.Log(job.String())
	t.Log("--------------------------------")
	err2 := jm.DeleteJob(context.TODO(), job.Name, job.Namespace)
	if err2 != nil {
		t.Log(err2.Error())
	}
	//time.Sleep(time.Millisecond * 1500)
	err3 := jm.CreateJob(context.TODO(), job)
	if err3 != nil {
		t.Log(err3.Error())
	}
}

func raw2Request(raw string) admission.Request {
	return admission.Request{
		AdmissionRequest: v1.AdmissionRequest{
			Object: runtime.RawExtension{
				Raw:    []byte(raw),
				Object: nil,
			},
		},
	}
}

func TestCompareImage(t *testing.T) {
	raw1 := `
apiVersion: batch/v1
kind: Job
metadata:
  annotations:
    job-refactor.sre.rootcloud.info/comparison-content: image
  labels:
    app.oam.dev/revision: test-job-v1
  name: test-job-latest
  namespace: default
spec:
  backoffLimit: 6
  completionMode: NonIndexed
  completions: 1
  parallelism: 1
  suspend: false
  template:
    metadata:
      creationTimestamp: null
      labels:
        app: test-job
        app.oam.dev/component: test-job
        job-name: test-job-latest
    spec:
      containers:
      - command:
        - /bin/sh
        - -c
        - |
          echo default
          sleep 10
        image: dockerhub.tencentcloudcr.com/library/nginx:v1
        imagePullPolicy: Always
        name: test-job
        resources:
          limits:
            cpu: 500m
            memory: 500Mi
          requests:
            cpu: 200m
            memory: 200Mi
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      dnsPolicy: ClusterFirst
      restartPolicy: Never
      schedulerName: default-scheduler
      terminationGracePeriodSeconds: 900
`
	raw2 := `
apiVersion: batch/v1
kind: Job
metadata:
  annotations:
    job-refactor.sre.rootcloud.info/comparison-content: image
  labels:
    app.oam.dev/revision: test-job-v2
  name: test-job-latest
  namespace: default
spec:
  backoffLimit: 6
  completionMode: NonIndexed
  completions: 1
  parallelism: 1
  suspend: false
  template:
    metadata:
      creationTimestamp: null
      labels:
        app: test-job
        app.oam.dev/component: test-job
        job-name: test-job-latest
    spec:
      containers:
      - command:
        - /bin/sh
        - -c
        - |
          echo default
          sleep 10
        env:  # different value, but not compare
        - name: k1
          value: v1
        - name: k2
          value: v22222 
        image: dockerhub.tencentcloudcr.com/library/nginx:v1  # same value
        imagePullPolicy: Always
        name: test-job
        resources:
          limits:
            cpu: 500m
            memory: 500Mi
          requests:
            cpu: 200m
            memory: 200Mi
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      dnsPolicy: ClusterFirst
      restartPolicy: Never
      schedulerName: default-scheduler
      terminationGracePeriodSeconds: 900
`
	raw3 := `
apiVersion: batch/v1
kind: Job
metadata:
  annotations:
    job-refactor.sre.rootcloud.info/comparison-content: image
  labels:
    app.oam.dev/revision: test-job-v3
  name: test-job-latest
  namespace: default
spec:
  backoffLimit: 6
  completionMode: NonIndexed
  completions: 1
  parallelism: 1
  suspend: false
  template:
    metadata:
      creationTimestamp: null
      labels:
        app: test-job
        app.oam.dev/component: test-job
        job-name: test-job-latest
    spec:
      containers:
      - command:
        - /bin/sh
        - -c
        - |
          echo default
          sleep 10
        env:  # different value, but not compare
        - name: k2
          value: v2123412341234234  # difference
        image: dockerhub.tencentcloudcr.com/library/nginx:v2 #different image
        imagePullPolicy: Always
        name: test-job
        resources:
          limits:
            cpu: 500m
            memory: 500Mi
          requests:
            cpu: 200m
            memory: 200Mi
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      dnsPolicy: ClusterFirst
      restartPolicy: Never
      schedulerName: default-scheduler
      terminationGracePeriodSeconds: 900
`
	jm := newJm()
	j1 := &batchv1.Job{}
	j2 := &batchv1.Job{}
	j3 := &batchv1.Job{}
	_ = jm.decoder.Decode(raw2Request(raw1), j1)
	_ = jm.decoder.Decode(raw2Request(raw2), j2)
	_ = jm.decoder.Decode(raw2Request(raw3), j3)
	assert.True(t, jm.CompareJob(j1, j2))
	assert.False(t, jm.CompareJob(j1, j3))
}

func TestCompareImageAndEnv(t *testing.T) {
	raw1 := `
apiVersion: batch/v1
kind: Job
metadata:
  annotations:
    job-refactor.sre.rootcloud.info/comparison-content: image,env
  labels:
    app.oam.dev/revision: test-job-v1
  name: test-job-latest
  namespace: default
spec:
  backoffLimit: 6
  completionMode: NonIndexed
  completions: 1
  parallelism: 1
  suspend: false
  template:
    metadata:
      creationTimestamp: null
      labels:
        app: test-job
        app.oam.dev/component: test-job
        job-name: test-job-latest
    spec:
      containers:
      - command:
        - /bin/sh
        - -c
        - |
          echo default
          sleep 10
        image: dockerhub.tencentcloudcr.com/library/nginx:v1
        imagePullPolicy: Always
        name: test-job
        resources:
          limits:
            cpu: 500m
            memory: 500Mi
          requests:
            cpu: 200m
            memory: 200Mi
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      dnsPolicy: ClusterFirst
      restartPolicy: Never
      schedulerName: default-scheduler
      terminationGracePeriodSeconds: 900
`
	raw2 := `
apiVersion: batch/v1
kind: Job
metadata:
  annotations:
    job-refactor.sre.rootcloud.info/comparison-content: image,env
  labels:
    app.oam.dev/revision: test-job-v2
  name: test-job-latest
  namespace: default
spec:
  backoffLimit: 6
  completionMode: NonIndexed
  completions: 1
  parallelism: 1
  suspend: false
  template:
    metadata:
      creationTimestamp: null
      labels:
        app: test-job
        app.oam.dev/component: test-job
        job-name: test-job-latest
    spec:
      containers:
      - command:
        - /bin/sh
        - -c
        - |
          echo default
          sleep 10
        env:  # different value
        - name: k1
          value: v1
        - name: k2
          value: v22222 
        image: dockerhub.tencentcloudcr.com/library/nginx:v1  # same value
        imagePullPolicy: Always
        name: test-job
        resources:
          limits:
            cpu: 500m
            memory: 500Mi
          requests:
            cpu: 200m
            memory: 200Mi
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      dnsPolicy: ClusterFirst
      restartPolicy: Never
      schedulerName: default-scheduler
      terminationGracePeriodSeconds: 900
`
	raw3 := `
apiVersion: batch/v1
kind: Job
metadata:
  annotations:
    job-refactor.sre.rootcloud.info/comparison-content: image,env
  labels:
    app.oam.dev/revision: test-job-v3
  name: test-job-latest
  namespace: default
spec:
  backoffLimit: 6
  completionMode: NonIndexed
  completions: 1
  parallelism: 1
  suspend: false
  template:
    metadata:
      creationTimestamp: null
      labels:
        app: test-job
        app.oam.dev/component: test-job
        job-name: test-job-latest
    spec:
      containers:
      - command:
        - /bin/sh
        - -c
        - |
          echo default
          sleep 10
        env:  # different value
        - name: k2
          value: v2123412341234234  # difference
        image: dockerhub.tencentcloudcr.com/library/nginx:v2 #different image
        imagePullPolicy: Always
        name: test-job
        resources:
          limits:
            cpu: 500m
            memory: 500Mi
          requests:
            cpu: 200m
            memory: 200Mi
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      dnsPolicy: ClusterFirst
      restartPolicy: Never
      schedulerName: default-scheduler
      terminationGracePeriodSeconds: 900
`
	jm := newJm()
	j1 := &batchv1.Job{}
	j2 := &batchv1.Job{}
	j3 := &batchv1.Job{}
	_ = jm.decoder.Decode(raw2Request(raw1), j1)
	_ = jm.decoder.Decode(raw2Request(raw2), j2)
	_ = jm.decoder.Decode(raw2Request(raw3), j3)
	assert.False(t, jm.CompareJob(j1, j2))
	assert.False(t, jm.CompareJob(j1, j3))
}

func TestCompareEnv(t *testing.T) {
	raw1 := `
apiVersion: batch/v1
kind: Job
metadata:
  annotations:
    job-refactor.sre.rootcloud.info/comparison-content: env
  labels:
    app.oam.dev/revision: test-job-v1
  name: test-job-latest
  namespace: default
spec:
  backoffLimit: 6
  completionMode: NonIndexed
  completions: 1
  parallelism: 1
  suspend: false
  template:
    metadata:
      creationTimestamp: null
      labels:
        app: test-job
        app.oam.dev/component: test-job
        job-name: test-job-latest
    spec:
      containers:
      - command:
        - /bin/sh
        - -c
        - |
          echo default
          sleep 10
        image: dockerhub.tencentcloudcr.com/library/nginx:v1
        imagePullPolicy: Always
        name: test-job
        resources:
          limits:
            cpu: 500m
            memory: 500Mi
          requests:
            cpu: 200m
            memory: 200Mi
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      dnsPolicy: ClusterFirst
      restartPolicy: Never
      schedulerName: default-scheduler
      terminationGracePeriodSeconds: 900
`
	raw2 := `
apiVersion: batch/v1
kind: Job
metadata:
  annotations:
    job-refactor.sre.rootcloud.info/comparison-content: env
  labels:
    app.oam.dev/revision: test-job-v2
  name: test-job-latest
  namespace: default
spec:
  backoffLimit: 6
  completionMode: NonIndexed
  completions: 1
  parallelism: 1
  suspend: false
  template:
    metadata:
      creationTimestamp: null
      labels:
        app: test-job
        app.oam.dev/component: test-job
        job-name: test-job-latest
    spec:
      containers:
      - command:
        - /bin/sh
        - -c
        - |
          echo default
          sleep 10
        env:  # different value
        - name: k1
          value: v1
        - name: k2
          value: v22222 
        image: dockerhub.tencentcloudcr.com/library/nginx:v1  # same value
        imagePullPolicy: Always
        name: test-job
        resources:
          limits:
            cpu: 500m
            memory: 500Mi
          requests:
            cpu: 200m
            memory: 200Mi
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      dnsPolicy: ClusterFirst
      restartPolicy: Never
      schedulerName: default-scheduler
      terminationGracePeriodSeconds: 900
`
	raw3 := `
apiVersion: batch/v1
kind: Job
metadata:
  annotations:
    job-refactor.sre.rootcloud.info/comparison-content: image,env
  labels:
    app.oam.dev/revision: test-job-v3
  name: test-job-latest
  namespace: default
spec:
  backoffLimit: 6
  completionMode: NonIndexed
  completions: 1
  parallelism: 1
  suspend: false
  template:
    metadata:
      creationTimestamp: null
      labels:
        app: test-job
        app.oam.dev/component: test-job
        job-name: test-job-latest
    spec:
      containers:
      - command:
        - /bin/sh
        - -c
        - |
          echo default
          sleep 10
        image: dockerhub.tencentcloudcr.com/library/nginx:65535 #different image
        imagePullPolicy: Always
        name: test-job
        resources:
          limits:
            cpu: 500m
            memory: 500Mi
          requests:
            cpu: 200m
            memory: 200Mi
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      dnsPolicy: ClusterFirst
      restartPolicy: Never
      schedulerName: default-scheduler
      terminationGracePeriodSeconds: 900
`
	jm := newJm()
	j1 := &batchv1.Job{}
	j2 := &batchv1.Job{}
	j3 := &batchv1.Job{}
	_ = jm.decoder.Decode(raw2Request(raw1), j1)
	_ = jm.decoder.Decode(raw2Request(raw2), j2)
	_ = jm.decoder.Decode(raw2Request(raw3), j3)
	assert.False(t, jm.CompareJob(j1, j2))
	assert.True(t, jm.CompareJob(j1, j3))
}
