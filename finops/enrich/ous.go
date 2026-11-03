package enrich

import (
	"fmt"

	"github.com/PopGreenTh/OpenXOps/record"
	"github.com/PopGreenTh/OpenXOps/record/taxonomy"
)

// ouEffect is what the taxonomy sets on one discovered OU: its Apps and its Environment (D-061).
type ouEffect struct {
	apps    []appPick
	appPath string // the OU's discovered path, for the app rule `ou:<path>`
	env     string // Environment code, "" when the OU sets none
	envPath string
}

// acctEntry is an account entry of the taxonomy with the OU entry it sits under.
type acctEntry struct {
	acc     *taxonomy.Account
	ouID    string // the discovered OU the entry's OU matches, "" when it matches none
	ouPath  string
	ouFound bool
}

// lzView is the taxonomy's LZ matched against the discovered OUs.
type lzView struct {
	effects  map[string]ouEffect // by discovered OU id
	accounts map[string]acctEntry
}

// matchOU finds the discovered OU an `ou_paths` entry names: the one whose id equals ref_id when
// the entry has one, else the one whose path equals path (D-061).
func (in inv) matchOU(ou taxonomy.OUPath) (id, path string, ok bool) {
	if ou.RefID != "" {
		info, ok := in.ous[ou.RefID]
		return info.id, info.path, ok
	}
	id, ok = in.byPath[ou.Path]
	return id, ou.Path, ok
}

// buildViews matches every LZ's OU entries to the discovered OUs. An entry that matches none gets
// an `ou-not-found` finding and is skipped; the sources after it still apply.
func (r *resolver) buildViews(findings *[]record.Record) {
	r.views = map[string]*lzView{}
	for i := range r.t.LZs {
		lz := &r.t.LZs[i]
		v := &lzView{effects: map[string]ouEffect{}, accounts: map[string]acctEntry{}}
		r.views[lz.Code] = v
		for j := range lz.OUPaths {
			ou := lz.OUPaths[j]
			id, path, ok := r.in.matchOU(ou)
			if !ok {
				subject := ou.Path
				if subject == "" {
					subject = ou.RefID
				}
				*findings = append(*findings, record.Finding("warning", "ou-not-found", subject,
					fmt.Sprintf("OU %q not found in inventory", subject), cmdMap))
			} else {
				eff := v.effects[id]
				if apps := r.apps(ou.RefURNApps); len(apps) > 0 {
					eff.apps, eff.appPath = apps, path
				}
				if env, _, ok := r.env(ou.RefURNEnv); ok {
					eff.env, eff.envPath = env, path
				}
				v.effects[id] = eff
			}
			for k := range ou.Accounts {
				a := &ou.Accounts[k]
				if _, dup := v.accounts[a.RefID]; !dup {
					v.accounts[a.RefID] = acctEntry{acc: a, ouID: id, ouPath: path, ouFound: ok}
				}
			}
		}
	}
}

// staleAccountEntries reports the account entries whose account was not discovered.
func (r *resolver) staleAccountEntries() []record.Record {
	var out []record.Record
	for i := range r.t.LZs {
		for _, ou := range r.t.LZs[i].OUPaths {
			for _, a := range ou.Accounts {
				if _, ok := r.in.accounts[a.RefID]; !ok {
					out = append(out, record.Finding("warning", "account-not-discovered", a.RefID,
						fmt.Sprintf("taxonomy account entry %s not in inventory", a.RefID), cmdMap))
				}
			}
		}
	}
	return out
}

// CheckCollisions returns an error when two OUs, or two accounts, of the inventory get one URN
// (D-059): an OU's URN holds its code and no parent, so two unlisted OUs both named `Prod` collide.
// The error names both OU paths or both account ids. The taxonomy breaks a tie by listing one of
// them with a `code` of its own.
func CheckCollisions(t taxonomy.Taxonomy, recs []record.Record) error {
	r := &resolver{t: t, in: buildInventory(recs)}
	lzOf := func(rec record.Record) string {
		if lz, ok := r.lzByMgmt(strID(rec["management_account"])); ok {
			return lz.Code
		}
		return lzUnalloc
	}
	ous := map[string]string{} // URN -> path of the first OU that has it
	for _, rec := range recs {
		if rec.Kind() != "ou" {
			continue
		}
		lz := lzOf(rec)
		path := strID(rec["path"])
		u := t.Cloud(lz).OUURN(t.LZURN(lz), taxonomy.DiscoveredOU{ID: strID(rec["id"]), Path: path, Name: strID(rec["name"])}).String()
		if first, dup := ous[u]; dup && first != path {
			return fmt.Errorf("OUs %s and %s both get the URN %s; list one of them under ou_paths with a code of its own", first, path, u)
		}
		ous[u] = path
	}
	accounts := map[string]string{} // URN -> id of the first account that has it
	for _, ac := range sortedAccounts(r.in.accounts) {
		_, u := r.cloudURNs(lzOf(ac), ac)
		id := strID(ac["id"])
		if first, dup := accounts[u]; dup {
			return fmt.Errorf("accounts %s and %s both get the URN %s; list one of them under accounts with a code of its own", first, id, u)
		}
		accounts[u] = id
	}
	return nil
}
