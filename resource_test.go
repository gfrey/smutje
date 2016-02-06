package smutje

import "testing"

func TestHandleChild(t *testing.T) {
	res, err := ReadFile("testdata/test_handle_child_base.smd")
	if err != nil {
		t.Fatalf("didn't expect an error, got: %s", err)
	}

	tt := []struct {
		got interface{}
		exp interface{}
		msg string
	}{
		{res.ID, "handle_child", "validate resource name"},
		{res.Attributes["HC_Key"], "1", "attrs: in host available"},
		{len(res.Packages), 3, "3 packages expected"},

		{res.Packages[0].ID, "inc.ipkg1", "inc pkg1 identifier"},
		{res.Packages[0].Attributes["HC_Key"], "", "base key (will be merged in on provisioning)"},
		{res.Packages[0].Attributes["HCI_Key"], "2", "attrs: include"},
		{res.Packages[0].Attributes["HCT_Key"], "3", "attrs: template"},
		{res.Packages[0].Attributes["HCT_Pkg_Key"], "4", "attrs: template pkg"},

		{res.Packages[1].ID, "inc.ipkg2", "inc pkg2 identifier"},
		{res.Packages[1].Attributes["HC_Key"], "", "base key (will be merged in on provisioning)"},
		{res.Packages[1].Attributes["HCI_Key"], "2", "attrs: include"},
		{res.Packages[1].Attributes["HCT_Key"], "3", "attrs: template"},
		{res.Packages[1].Attributes["HCT_Pkg_Key"], "5", "attrs: template pkg"},

		{res.Packages[2].ID, "pkg3", "pkg3 identifier"},
		{res.Packages[2].Attributes["HC_Key"], "", "base key (will be merged in on provisioning)"},
		{res.Packages[2].Attributes["HCI_Key"], "", "include attrs must not be available"},
		{res.Packages[2].Attributes["HCT_Key"], "", "template attrs must not be available"},
		{res.Packages[2].Attributes["HCT_Pkg_Key"], "", "template pkg attrs must not be available"},
	}

	for _, tti := range tt {
		if tti.got != tti.exp {
			t.Errorf("%#v [got] != %#v [exp] (%s)", tti.got, tti.exp, tti.msg)
		}
	}
}
