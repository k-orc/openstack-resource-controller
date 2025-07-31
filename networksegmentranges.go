
package networksegmentranges

import (
	"github.com/gophercloud/gophercloud"
)

type NetworkSegmentRange struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	NetworkType     string   `json:"network_type"`
	PhysicalNetwork string   `json:"physical_network"`
	Minimum         int      `json:"minimum"`
	Maximum         int      `json:"maximum"`
	Shared          bool     `json:"shared"`
	Default         bool     `json:"default"`
	ProjectID       string   `json:"project_id"`
	Tags            []string `json:"tags"`
}

type CreateOpts struct {
	Name            string   `json:"name" required:"true"`
	NetworkType     string   `json:"network_type" required:"true"`
	PhysicalNetwork string   `json:"physical_network" required:"true"`
	Minimum         int      `json:"minimum" required:"true"`
	Maximum         int      `json:"maximum" required:"true"`
	Shared          *bool    `json:"shared,omitempty"`
	Default         *bool    `json:"default,omitempty"`
	ProjectID       string   `json:"project_id,omitempty"`
	Tags            []string `json:"tags,omitempty"`
}
type UpdateOpts struct {
	Name            *string   `json:"name,omitempty"`
	NetworkType     *string   `json:"network_type,omitempty"`
	PhysicalNetwork *string   `json:"physical_network,omitempty"`
	Minimum         *int      `json:"minimum,omitempty"`
	Maximum         *int      `json:"maximum,omitempty"`
	Shared          *bool     `json:"shared,omitempty"`
	Default         *bool     `json:"default,omitempty"`
	Tags            *[]string `json:"tags,omitempty"`
}

type ListOpts struct {
	Name string `q:"name"`
}

type commonResult struct {
	gophercloud.Result
}

type CreateResult struct{ commonResult }
type GetResult struct{ commonResult }
type UpdateResult struct{ commonResult }
type DeleteResult struct{ gophercloud.ErrResult }
type ListResult struct{ gophercloud.Result }


func rootURL(c *gophercloud.ServiceClient) string {
	return c.ServiceURL("network_segment_ranges")
}
func resourceURL(c *gophercloud.ServiceClient, id string) string {
	return c.ServiceURL("network_segment_ranges", id)
}

// --- CRUD OPS ---
func Create(c *gophercloud.ServiceClient, opts CreateOpts) (r CreateResult) {
	b, err := gophercloud.BuildRequestBody(opts, "network_segment_range")
	if err != nil {
		r.Err = err
		return
	}
	_, r.Err = c.Post(rootURL(c), b, &r.Body, nil)
	return
}

// GET one by ID
func Get(c *gophercloud.ServiceClient, id string) (r GetResult) {
	_, r.Err = c.Get(resourceURL(c, id), &r.Body, nil)
	return
}

// PUT update
func Update(c *gophercloud.ServiceClient, id string, opts UpdateOpts) (r UpdateResult) {
	b, err := gophercloud.BuildRequestBody(opts, "network_segment_range")
	if err != nil {
		r.Err = err
		return
	}
	_, r.Err = c.Put(resourceURL(c, id), b, &r.Body, nil)
	return
}

// DELETE
func Delete(c *gophercloud.ServiceClient, id string) (r DeleteResult) {
	_, r.Err = c.Delete(resourceURL(c, id), nil)
	return
}

// GET (list all)
func List(c *gophercloud.ServiceClient, opts ListOpts) pagination.Pager {
	return pagination.NewPager(c, rootURL(c), func(r pagination.PageResult) pagination.Page {
		return NetworkSegmentRangePage{pagination.LinkedPageBase{PageResult: r}}
	})
}

// pagination
type NetworkSegmentRangePage struct {
	pagination.LinkedPageBase
}

func (r NetworkSegmentRangePage) IsEmpty() (bool, error) {
	objs, err := ExtractNetworkSegmentRanges(r)
	return len(objs) == 0, err
}

func ExtractNetworkSegmentRanges(r pagination.Page) ([]NetworkSegmentRange, error) {
	var s struct {
		NetworkSegmentRanges []NetworkSegmentRange `json:"network_segment_ranges"`
	}
	err := (r.(NetworkSegmentRangePage)).ExtractInto(&s)
	return s.NetworkSegmentRanges, err
}


