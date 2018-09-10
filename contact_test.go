package contacts

import (
	"testing"
	"time"
)

func TestAttributeChanges(t *testing.T) {
	c := &Contact{
		ID:       "id",
		Name:     "Name",
		First:    "First",
		Last:     "Last",
		Birthday: time.Now(),
		Email:    []string{"Email1", "Email2"},
		Phone:    []string{"Phone", "Phone Alt"},
		Labels:   []string{"Label", "Another Label"},
	}
	vals := c.attributeValues()
	if len(vals) != 8 {
		t.Errorf("vals was not expected length: %+v", vals)
	}
	c1 := &Contact{
		ID:       "id2",
		Name:     "Name",
		Last:     "Last",
		Birthday: time.Now(),
		Email:    []string{"Email1"},
		Phone:    []string{"Phone", "Phone Alt", "Phone 3"},
		Labels:   []string{"Label", "Another Label"},
	}
	ch := c.changes(c1)
	t.Logf("changes: %+v", ch)
	if len(ch["add"]) != 0 {
		t.Errorf("wrong number of adds: %+v", ch["add"])
	}
	if len(ch["delete"]) != 1 {
		t.Errorf("wrong number of deletes: %+v", ch["delete"])
	}
	if len(ch["replace"]) != 2 {
		t.Errorf("wrong number of replaces: %+v", ch["replace"])
	}
	if len(ch["modify"]) != 1 {
		t.Errorf("wrong number of modifies: %+v", ch["modify"])
	}
}
