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
		{res.Packages[0].Attributes["HC_Key"], "1", "base key is available everywhere"},
		{res.Packages[0].Attributes["HC_Overwritten_Key"], "3", "key's overwritten in packages rule out everything"},
		{res.Packages[0].Attributes["HCI_Key"], "4", "included keys are overwritten in package, too"},
		{res.Packages[0].Attributes["HCT_Key"], "3", "template attributes are available in template's packages"},
		{res.Packages[0].Attributes["HCT_Pkg_Key"], "4", "template package attributes are set properly"},

		{res.Packages[1].ID, "inc.ipkg2", "inc pkg2 identifier"},
		{res.Packages[1].Attributes["HC_Key"], "1", "base key is available everywhere"},
		{res.Packages[1].Attributes["HC_Overwritten_Key"], "4", "key's overwritten in packages rule out everything"},
		{res.Packages[1].Attributes["HCI_Key"], "2", "included keys are overwritten in package, too"},
		{res.Packages[1].Attributes["HCT_Key"], "3", "template attributes are available in template's packages"},
		{res.Packages[1].Attributes["HCT_Pkg_Key"], "5", "template package attributes are set properly"},

		{res.Packages[2].ID, "pkg3", "pkg3 identifier"},
		{res.Packages[2].Attributes["HC_Key"], "1", "base key is available everywhere"},
		{res.Packages[2].Attributes["HC_Pkg_Key"], "5", "keys in regular resource packages work"},
		{res.Packages[2].Attributes["HC_Overwritten_Key"], "2", "included keys are overwritten in package, too"},
		{res.Packages[2].Attributes["HCI_Key"], "", "include attrs must not be available"},
		{res.Packages[2].Attributes["HCT_Key"], "", "template attributes are available in template's packages, only"},
		{res.Packages[2].Attributes["HCT_Pkg_Key"], "", "template pkg attrs must not be available"},
	}

	for i, tti := range tt {
		if tti.got != tti.exp {
			t.Errorf("%d: %#v [got] != %#v [exp] (%s)", i, tti.got, tti.exp, tti.msg)
		}
	}
}
