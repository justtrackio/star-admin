package internal

import "testing"

func TestDictionaryMetadata(t *testing.T) {
	handler := &HandlerDictionaries{}

	for _, columns := range []map[string]string{
		{"dictionaryObject": "lookup_apps", "status": "FINISHED"},
		{"dictionarySource": "lookup_apps", "state": "FINISHED"},
	} {
		columns["DictionaryId"] = "9007199254740993"
		columns["DictionaryName"] = "apps"
		columns["ErrorMessage"] = "Cancelled and rollback to previous state"
		columns["approximated dictionaryMemoryUsage (Bytes)"] = "Can not get memory info"

		dictionary := handler.dictionaryFromMetadata(columns)
		if dictionary.ID != "9007199254740993" || dictionary.Name != "apps" || dictionary.Source != "lookup_apps" || dictionary.Status != "FINISHED" {
			t.Fatalf("lost dictionary identity or column aliases: %+v", dictionary)
		}

		if dictionary.Error != columns["ErrorMessage"] || dictionary.MemoryUsage != columns["approximated dictionaryMemoryUsage (Bytes)"] {
			t.Fatalf("finished status must preserve refresh and memory errors: %+v", dictionary)
		}
	}
}

func TestDictionaryMissingMetadata(t *testing.T) {
	handler := &HandlerDictionaries{}

	dictionary := handler.dictionaryFromMetadata(map[string]string{"FutureColumn": "ignored"})
	if dictionary != (Dictionary{}) {
		t.Fatalf("missing metadata must remain unknown: %+v", dictionary)
	}
}
