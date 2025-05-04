package backend

import "go.hasen.dev/vbeam"

// global (but volatile) list of usernames
var families []string

type AddFamilyRequest struct {
	Name string
}

type FamilyListResponse struct {
	AllFamilyNames []string
}

func AddFamily(ctx *vbeam.Context, req AddFamilyRequest) (resp FamilyListResponse, err error) {
	families = append(families, req.Name)
	resp.AllFamilyNames = families
	return
}

type Empty struct{}

func ListFamilies(ctx *vbeam.Context, req Empty) (resp FamilyListResponse, err error) {
	if families == nil {
		families = make([]string, 0)
	}
	resp.AllFamilyNames = families
	return
}
