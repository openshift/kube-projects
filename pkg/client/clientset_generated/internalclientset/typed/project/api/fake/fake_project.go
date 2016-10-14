package fake

import (
	api "github.com/openshift/kube-projects/pkg/project/api"
	pkg_api "k8s.io/kubernetes/pkg/api"
	unversioned "k8s.io/kubernetes/pkg/api/unversioned"
	v1 "k8s.io/kubernetes/pkg/api/v1"
	core "k8s.io/kubernetes/pkg/client/testing/core"
	labels "k8s.io/kubernetes/pkg/labels"
	watch "k8s.io/kubernetes/pkg/watch"
)

// FakeProjects implements ProjectInterface
type FakeProjects struct {
	Fake *FakeProject
	ns   string
}

var projectsResource = unversioned.GroupVersionResource{Group: "project", Version: "api", Resource: "projects"}

func (c *FakeProjects) Create(project *api.Project) (result *api.Project, err error) {
	obj, err := c.Fake.
		Invokes(core.NewCreateAction(projectsResource, c.ns, project), &api.Project{})

	if obj == nil {
		return nil, err
	}
	return obj.(*api.Project), err
}

func (c *FakeProjects) Update(project *api.Project) (result *api.Project, err error) {
	obj, err := c.Fake.
		Invokes(core.NewUpdateAction(projectsResource, c.ns, project), &api.Project{})

	if obj == nil {
		return nil, err
	}
	return obj.(*api.Project), err
}

func (c *FakeProjects) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(core.NewDeleteAction(projectsResource, c.ns, name), &api.Project{})

	return err
}

func (c *FakeProjects) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := core.NewDeleteCollectionAction(projectsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &api.ProjectList{})
	return err
}

func (c *FakeProjects) Get(name string) (result *api.Project, err error) {
	obj, err := c.Fake.
		Invokes(core.NewGetAction(projectsResource, c.ns, name), &api.Project{})

	if obj == nil {
		return nil, err
	}
	return obj.(*api.Project), err
}

func (c *FakeProjects) List(opts v1.ListOptions) (result *api.ProjectList, err error) {
	obj, err := c.Fake.
		Invokes(core.NewListAction(projectsResource, c.ns, opts), &api.ProjectList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := core.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &api.ProjectList{}
	for _, item := range obj.(*api.ProjectList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested projects.
func (c *FakeProjects) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(core.NewWatchAction(projectsResource, c.ns, opts))

}

// Patch applies the patch and returns the patched project.
func (c *FakeProjects) Patch(name string, pt pkg_api.PatchType, data []byte, subresources ...string) (result *api.Project, err error) {
	obj, err := c.Fake.
		Invokes(core.NewPatchSubresourceAction(projectsResource, c.ns, name, data, subresources...), &api.Project{})

	if obj == nil {
		return nil, err
	}
	return obj.(*api.Project), err
}
