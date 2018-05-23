// Copyright (C) 2015 Alex Sergeyev
// This project is licensed under the terms of the MIT license.
// Read LICENSE file for information for all notices and permissions.

package nradix

import "testing"

func TestTree(t *testing.T) {
	tr := NewTree(0)
	if tr == nil || tr.root == nil {
		t.Error("Did not create tree properly")
	}
	err := tr.AddCIDR("1.2.3.0/25", 1)
	if err != nil {
		t.Error(err)
	}

	// Matching defined cidr
	inf, err := tr.FindCIDR("1.2.3.1/25")
	if err != nil {
		t.Error(err)
	}
	if inf.(int) != 1 {
		t.Errorf("Wrong value, expected 1, got %v", inf)
	}

	// Inside defined cidr
	inf, err = tr.FindCIDR("1.2.3.60/32")
	if err != nil {
		t.Error(err)
	}
	if inf.(int) != 1 {
		t.Errorf("Wrong value, expected 1, got %v", inf)
	}
	inf, err = tr.FindCIDR("1.2.3.60")
	if err != nil {
		t.Error(err)
	}
	if inf.(int) != 1 {
		t.Errorf("Wrong value, expected 1, got %v", inf)
	}

	// Outside defined cidr
	inf, err = tr.FindCIDR("1.2.3.160/32")
	if err != nil {
		t.Error(err)
	}
	if inf != nil {
		t.Errorf("Wrong value, expected nil, got %v", inf)
	}
	inf, err = tr.FindCIDR("1.2.3.160")
	if err != nil {
		t.Error(err)
	}
	if inf != nil {
		t.Errorf("Wrong value, expected nil, got %v", inf)
	}

	inf, err = tr.FindCIDR("1.2.3.128/25")
	if err != nil {
		t.Error(err)
	}
	if inf != nil {
		t.Errorf("Wrong value, expected nil, got %v", inf)
	}

	// Covering not defined
	inf, err = tr.FindCIDR("1.2.3.0/24")
	if err != nil {
		t.Error(err)
	}
	if inf != nil {
		t.Errorf("Wrong value, expected nil, got %v", inf)
	}

	// Covering defined
	err = tr.AddCIDR("1.2.3.0/24", 2)
	if err != nil {
		t.Error(err)
	}
	inf, err = tr.FindCIDR("1.2.3.0/24")
	if err != nil {
		t.Error(err)
	}
	if inf.(int) != 2 {
		t.Errorf("Wrong value, expected 2, got %v", inf)
	}

	inf, err = tr.FindCIDR("1.2.3.160/32")
	if err != nil {
		t.Error(err)
	}
	if inf.(int) != 2 {
		t.Errorf("Wrong value, expected 2, got %v", inf)
	}

	// Hit both covering and internal, should choose most specific
	inf, err = tr.FindCIDR("1.2.3.0/32")
	if err != nil {
		t.Error(err)
	}
	if inf.(int) != 1 {
		t.Errorf("Wrong value, expected 1, got %v", inf)
	}

	// Delete internal
	err = tr.DeleteCIDR("1.2.3.0/25")
	if err != nil {
		t.Error(err)
	}

	// Hit covering with old IP
	inf, err = tr.FindCIDR("1.2.3.0/32")
	if err != nil {
		t.Error(err)
	}
	if inf.(int) != 2 {
		t.Errorf("Wrong value, expected 2, got %v", inf)
	}

	// Add internal back in
	err = tr.AddCIDR("1.2.3.0/25", 1)
	if err != nil {
		t.Error(err)
	}

	// Delete covering
	err = tr.DeleteCIDR("1.2.3.0/24")
	if err != nil {
		t.Error(err)
	}

	// Hit with old IP
	inf, err = tr.FindCIDR("1.2.3.0/32")
	if err != nil {
		t.Error(err)
	}
	if inf.(int) != 1 {
		t.Errorf("Wrong value, expected 1, got %v", inf)
	}

	// Find covering again
	inf, err = tr.FindCIDR("1.2.3.0/24")
	if err != nil {
		t.Error(err)
	}
	if inf != nil {
		t.Errorf("Wrong value, expected nil, got %v", inf)
	}

	// Add covering back in
	err = tr.AddCIDR("1.2.3.0/24", 2)
	if err != nil {
		t.Error(err)
	}
	inf, err = tr.FindCIDR("1.2.3.0/24")
	if err != nil {
		t.Error(err)
	}
	if inf.(int) != 2 {
		t.Errorf("Wrong value, expected 2, got %v", inf)
	}

	// Delete the whole range
	err = tr.DeleteWholeRangeCIDR("1.2.3.0/24")
	if err != nil {
		t.Error(err)
	}
	// should be no value for covering
	inf, err = tr.FindCIDR("1.2.3.0/24")
	if err != nil {
		t.Error(err)
	}
	if inf != nil {
		t.Errorf("Wrong value, expected nil, got %v", inf)
	}
	// should be no value for internal
	inf, err = tr.FindCIDR("1.2.3.0/32")
	if err != nil {
		t.Error(err)
	}
	if inf != nil {
		t.Errorf("Wrong value, expected nil, got %v", inf)
	}
}

func TestSet(t *testing.T) {
	tr := NewTree(0)
	if tr == nil || tr.root == nil {
		t.Error("Did not create tree properly")
	}

	tr.AddCIDR("1.1.1.0/24", 1)
	inf, err := tr.FindCIDR("1.1.1.0")
	if err != nil {
		t.Error(err)
	}
	if inf.(int) != 1 {
		t.Errorf("Wrong value, expected 1, got %v", inf)
	}

	tr.AddCIDR("1.1.1.0/25", 2)
	inf, err = tr.FindCIDR("1.1.1.0")
	if err != nil {
		t.Error(err)
	}
	if inf.(int) != 2 {
		t.Errorf("Wrong value, expected 2, got %v", inf)
	}
	inf, err = tr.FindCIDR("1.1.1.0/24")
	if err != nil {
		t.Error(err)
	}
	if inf.(int) != 1 {
		t.Errorf("Wrong value, expected 1, got %v", inf)
	}

	// add covering should fail
	err = tr.AddCIDR("1.1.1.0/24", 60)
	if err != ErrNodeBusy {
		t.Errorf("Should have gotten ErrNodeBusy, instead got err: %v", err)
	}

	// set covering
	err = tr.SetCIDR("1.1.1.0/24", 3)
	if err != nil {
		t.Error(err)
	}
	inf, err = tr.FindCIDR("1.1.1.0")
	if err != nil {
		t.Error(err)
	}
	if inf.(int) != 2 {
		t.Errorf("Wrong value, expected 2, got %v", inf)
	}
	inf, err = tr.FindCIDR("1.1.1.0/24")
	if err != nil {
		t.Error(err)
	}
	if inf.(int) != 3 {
		t.Errorf("Wrong value, expected 3, got %v", inf)
	}

	// set internal
	err = tr.SetCIDR("1.1.1.0/25", 4)
	if err != nil {
		t.Error(err)
	}
	inf, err = tr.FindCIDR("1.1.1.0")
	if err != nil {
		t.Error(err)
	}
	if inf.(int) != 4 {
		t.Errorf("Wrong value, expected 4, got %v", inf)
	}
	inf, err = tr.FindCIDR("1.1.1.0/24")
	if err != nil {
		t.Error(err)
	}
	if inf.(int) != 3 {
		t.Errorf("Wrong value, expected 3, got %v", inf)
	}
}

func TestRegression(t *testing.T) {
	tr := NewTree(0)
	if tr == nil || tr.root == nil {
		t.Error("Did not create tree properly")
	}

	tr.AddCIDR("1.1.1.0/24", 1)

	tr.DeleteCIDR("1.1.1.0/24")
	tr.AddCIDR("1.1.1.0/25", 2)

	// inside old range, outside new range
	inf, err := tr.FindCIDR("1.1.1.128")
	if err != nil {
		t.Error(err)
	} else if inf != nil {
		t.Errorf("Wrong value, expected nil, got %v", inf)
	}
}

func TestTree6(t *testing.T) {
	tr := NewTree(0)
	if tr == nil || tr.root == nil {
		t.Error("Did not create tree properly")
	}
	err := tr.AddCIDR("dead::0/16", 3)
	if err != nil {
		t.Error(err)
	}

	// Matching defined cidr
	inf, err := tr.FindCIDR("dead::beef")
	if err != nil {
		t.Error(err)
	}
	if inf.(int) != 3 {
		t.Errorf("Wrong value, expected 3, got %v", inf)
	}

	// Outside
	inf, err = tr.FindCIDR("deed::beef/32")
	if err != nil {
		t.Error(err)
	}
	if inf != nil {
		t.Errorf("Wrong value, expected nil, got %v", inf)
	}

	// Subnet
	err = tr.AddCIDR("dead:beef::0/48", 4)
	if err != nil {
		t.Error(err)
	}

	// Match defined subnet
	inf, err = tr.FindCIDR("dead:beef::0a5c:0/64")
	if err != nil {
		t.Error(err)
	}
	if inf.(int) != 4 {
		t.Errorf("Wrong value, expected 4, got %v", inf)
	}

	// Match outside defined subnet
	inf, err = tr.FindCIDR("dead:0::beef:0a5c:0/64")
	if err != nil {
		t.Error(err)
	}
	if inf.(int) != 3 {
		t.Errorf("Wrong value, expected 3, got %v", inf)
	}

}

func TestRegression6(t *testing.T) {
	tr := NewTree(0)
	if tr == nil || tr.root == nil {
		t.Error("Did not create tree properly")
	}
	// in one of the implementations /128 addresses were causing panic...
	tr.AddCIDR("2620:10f::/32", 54321)
	tr.AddCIDR("2620:10f:d000:100::5/128", 12345)

	inf, err := tr.FindCIDR("2620:10f:d000:100::5/128")
	if err != nil {
		t.Errorf("Could not get /128 address from the tree, error: %s", err)
	} else if inf.(int) != 12345 {
		t.Errorf("Wrong value from /128 test, got %d, expected 12345", inf)
	}
}
