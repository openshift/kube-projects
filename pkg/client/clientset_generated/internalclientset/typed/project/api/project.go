package api

import (
	api "github.com/openshift/kube-projects/pkg/project/api"
	pkg_api "k8s.io/kubernetes/pkg/api"
	v1 "k8s.io/kubernetes/pkg/api/v1"
	meta_v1 "k8s.io/kubernetes/pkg/apis/meta/v1"
	restclient "k8s.io/kubernetes/pkg/client/restclient"
	watch "k8s.io/kubernetes/pkg/watch"
)

// ProjectsGetter has a method to return a ProjectInterface.
// A group's client should implement this interface.
type ProjectsGetter interface {
	Projects(namespace string) ProjectInterface
}

// ProjectInterface has methods to work with Project resources.
type ProjectInterface interface {
	Create(*api.Project) (*api.Project, error)
	Update(*api.Project) (*api.Project, error)
	UpdateStatus(*api.Project) (*api.Project, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options meta_v1.GetOptions) (*api.Project, error)
	List(opts v1.ListOptions) (*api.ProjectList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt pkg_api.PatchType, data []byte, subresources ...string) (result *api.Project, err error)
	ProjectExpansion
}

// projects implements ProjectInterface
type projects struct {
	client restclient.Interface
	ns     string
}

// newProjects returns a Projects
func newProjects(c *ProjectApiClient, namespace string) *projects {
	return &projects{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Create takes the representation of a project and creates it.  Returns the server's representation of the project, and an error, if there is any.
func (c *projects) Create(project *api.Project) (result *api.Project, err error) {
	result = &api.Project{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("projects").
		Body(project).
		Do().
		Into(result)
	return
}

// Update takes the representation of a project and updates it. Returns the server's representation of the project, and an error, if there is any.
func (c *projects) Update(project *api.Project) (result *api.Project, err error) {
	result = &api.Project{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("projects").
		Name(project.Name).
		Body(project).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclientstatus=false comment above the type to avoid generating UpdateStatus().

func (c *projects) UpdateStatus(project *api.Project) (result *api.Project, err error) {
	result = &api.Project{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("projects").
		Name(project.Name).
		SubResource("status").
		Body(project).
		Do().
		Into(result)
	return
}

// Delete takes name of the project and deletes it. Returns an error if one occurs.
func (c *projects) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("projects").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *projects) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("projects").
		VersionedParams(&listOptions, pkg_api.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Get takes name of the project, and returns the corresponding project object, and an error if there is any.
func (c *projects) Get(name string, options meta_v1.GetOptions) (result *api.Project, err error) {
	result = &api.Project{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("projects").
		Name(name).
		VersionedParams(&options, pkg_api.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Projects that match those selectors.
func (c *projects) List(opts v1.ListOptions) (result *api.ProjectList, err error) {
	result = &api.ProjectList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("projects").
		VersionedParams(&opts, pkg_api.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested projects.
func (c *projects) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.client.Get().
		Prefix("watch").
		Namespace(c.ns).
		Resource("projects").
		VersionedParams(&opts, pkg_api.ParameterCodec).
		Watch()
}

// Patch applies the patch and returns the patched project.
func (c *projects) Patch(name string, pt pkg_api.PatchType, data []byte, subresources ...string) (result *api.Project, err error) {
	result = &api.Project{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("projects").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
