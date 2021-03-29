/*
Copyright 2021 The Knative Authors
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

package k8s

import (
	"fmt"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

// JobRunner will launch a Job and monitor it for completion.
type JobRunner interface {
	Run(*batchv1.Job) error
}

// CreateJobRunner will create a JobRunner, or return an error.
func CreateJobRunner(kube Clients) JobRunner {
	return &jobRunner{
		kube: kube,
	}
}

type jobRunner struct {
	kube Clients
}

func (j *jobRunner) Run(job *batchv1.Job) error {
	jobs := j.kube.Typed().BatchV1().Jobs(job.Namespace)
	_, err := jobs.Create(j.kube.Context(), job, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnexcpected, err)
	}
	factory := kubeinformers.NewSharedInformerFactoryWithOptions(
		j.kube.Typed(),
		time.Minute,
		kubeinformers.WithNamespace(job.Namespace),
		kubeinformers.WithTweakListOptions(func(options *metav1.ListOptions) {
			options.FieldSelector = fmt.Sprintf("metadata.name=%s", job.Name)
		}),
	)
	// FIXME: This function do not wait properly for the end of the Job
	stop := make(chan struct{})
	jobsInformer := factory.Batch().V1().Jobs().Informer()
	jobsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			close(stop)
		},
	})
	go factory.Start(stop)
	waitOnStop(stop)
	updated, err := jobs.Get(j.kube.Context(), job.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnexcpected, err)
	}
	if updated.Status.Succeeded < 1 {
		return fmt.Errorf("%w: %s", ErrUnexcpected, "expected to have successful job")
	}
	return jobs.Delete(j.kube.Context(), job.Name, metav1.DeleteOptions{})
}

func waitOnStop(stop chan struct{}) {
	for {
		select {
		case <-stop:
			return
		default:
			time.Sleep(time.Second)
		}
	}
}
