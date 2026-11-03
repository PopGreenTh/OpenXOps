package taxonomy

import (
	"strings"

	"github.com/PopGreenTh/OpenXOps/record/urn"
)

// URN builders (D-046, D-058): a URN is built from the codes, lowercased (D-056). A `urn` written
// in the file is only checked against these.

func (t Taxonomy) tenantURN() urn.URN { return urn.Tenant(t.Tenant.Code) }

// BUURN is the URN of a business unit.
func (t Taxonomy) BUURN(bu string) urn.URN { return t.tenantURN().Child("bzu", bu) }

// CCURN is the URN of a cost center, which sits under its own business unit.
func (t Taxonomy) CCURN(bu, cc string) urn.URN { return t.BUURN(bu).Child("cct", cc) }

// ProjectURN is the URN of a project.
func (t Taxonomy) ProjectURN(prj string) urn.URN { return t.tenantURN().Child("prj", prj) }

// AppURN is the URN of an App, which sits under its project.
func (t Taxonomy) AppURN(prj, app string) urn.URN { return t.ProjectURN(prj).Child("app", app) }

// EnvURN is the URN of an Environment, which sits under its App.
func (t Taxonomy) EnvURN(prj, app, env string) urn.URN { return t.AppURN(prj, app).Child("env", env) }

// LZURN is the URN of the landing zone.
func (t Taxonomy) LZURN(lz string) urn.URN { return t.tenantURN().Child("lzc", lz) }

// CCRef is a cost center with the business unit it sits under.
type CCRef struct {
	CC *CostCenter
	BU *BusinessUnit
}

// AppRef is an App with its project.
type AppRef struct {
	App     *App
	Project *Project
}

// EnvRef is an Environment with its App and project.
type EnvRef struct {
	Env     *Environment
	App     *App
	Project *Project
}

// Index finds the node a `ref_urn_*` value names. Keys are the URNs built from the codes, so a
// reference matches whatever its case.
type Index struct {
	BU  map[string]*BusinessUnit
	CC  map[string]CCRef
	Prj map[string]*Project
	App map[string]AppRef
	Env map[string]EnvRef

	byCode map[string]AppRef // Apps by lower-case code; Apps are unique in the tenant
}

func key(u string) string { return strings.ToLower(strings.TrimSpace(u)) }

// Index builds the index of t. Its pointers point into t's own slices.
func (t Taxonomy) Index() Index {
	ix := Index{
		BU: map[string]*BusinessUnit{}, CC: map[string]CCRef{}, Prj: map[string]*Project{},
		App: map[string]AppRef{}, Env: map[string]EnvRef{}, byCode: map[string]AppRef{},
	}
	for i := range t.BusinessUnits {
		bu := &t.BusinessUnits[i]
		ix.BU[t.BUURN(bu.Code).String()] = bu
		for j := range bu.CostCenters {
			cc := &bu.CostCenters[j]
			ix.CC[t.CCURN(bu.Code, cc.Code).String()] = CCRef{CC: cc, BU: bu}
		}
	}
	for i := range t.Projects {
		p := &t.Projects[i]
		ix.Prj[t.ProjectURN(p.Code).String()] = p
		for j := range p.Apps {
			a := &p.Apps[j]
			ref := AppRef{App: a, Project: p}
			ix.App[t.AppURN(p.Code, a.Code).String()] = ref
			if _, dup := ix.byCode[key(a.Code)]; !dup {
				ix.byCode[key(a.Code)] = ref
			}
			for k := range a.Environments {
				e := &a.Environments[k]
				ix.Env[t.EnvURN(p.Code, a.Code, e.Code).String()] = EnvRef{Env: e, App: a, Project: p}
			}
		}
	}
	return ix
}

// LookupBU finds the business unit a URN names.
func (ix Index) LookupBU(u string) (*BusinessUnit, bool) { b, ok := ix.BU[key(u)]; return b, ok }

// LookupCC finds the cost center a URN names.
func (ix Index) LookupCC(u string) (CCRef, bool) { c, ok := ix.CC[key(u)]; return c, ok }

// LookupApp finds the App a URN names.
func (ix Index) LookupApp(u string) (AppRef, bool) { a, ok := ix.App[key(u)]; return a, ok }

// LookupEnv finds the Environment a URN names.
func (ix Index) LookupEnv(u string) (EnvRef, bool) { e, ok := ix.Env[key(u)]; return e, ok }

// LookupProject finds the project a URN names.
func (ix Index) LookupProject(u string) (*Project, bool) { p, ok := ix.Prj[key(u)]; return p, ok }

// AppByCode finds an App by its code, in any case.
func (ix Index) AppByCode(code string) (AppRef, bool) { a, ok := ix.byCode[key(code)]; return a, ok }

// HasEnv reports whether the App defines an Environment with this code.
func (a App) HasEnv(code string) bool {
	for _, e := range a.Environments {
		if e.Code == code {
			return true
		}
	}
	return false
}
