package semver

import (
	"testing"
)

var (
	series = []string{
		"1.0.0-alpha",
		"1.0.0-alpha.1",
		"1.0.0-alpha.beta",
		"1.0.0-beta",
		"1.0.0-beta.2",
		"1.0.0-beta.11",
		"1.0.0-rc.1",
		"1.0.0",
	}
)

func testVersions(t *testing.T, version1, version2 string, result int) {
	t.Logf("Testing %q and %q, result should be %d", version1, version2, result)
	v1, err := Parse(version1)
	if err != nil {
		t.Error(err)
	}
	v2, err := Parse(version2)
	if err != nil {
		t.Error(err)
	}
	if v1 != nil && v2 != nil {
		if r := v1.Compare(v2); r != result {
			t.Errorf("Expected %d, got %d comparing %q and %q", result, r, version1, version2)
		}
	}
}

func TestEqual(t *testing.T) {
	testVersions(t, "1.0", "1.0", 0)
	testVersions(t, "1.0", "1.0+foo.bar.baz", 0)
	testVersions(t, "1.1", "1.1.0", 0)
	testVersions(t, "0.1", "0.1", 0)
}

func TestLess(t *testing.T) {
	testVersions(t, "0.9", "1.5", -1)
	testVersions(t, "1.0-alpha", "1.0", -1)
	testVersions(t, "1.0-rc1", "1.0", -1)
	testVersions(t, "1.0-rc1", "1.0-rc2", -1)
	testVersions(t, "1.0-1", "1.0-gamma", -1)
	testVersions(t, "1.3.7", "1.5", -1)
	testVersions(t, "1.0.5", "1.5", -1)
}

func TestSeries(t *testing.T) {
	versions := make([]*Version, len(series))
	for ii, v := range series {
		ver, err := Parse(v)
		if err != nil {
			t.Fatal(err)
		}
		versions[ii] = ver
	}
	for ii, v := range versions {
		for jj := ii + 1; jj < len(versions); jj++ {
			if !v.Lower(versions[jj]) {
				t.Errorf("%q should be lower than %q", series[ii], series[jj])
			}
		}
	}
}
