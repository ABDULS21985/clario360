package contradiction

import (
	"strings"

	"github.com/clario360/platform/internal/data/model"
	cruntime "github.com/clario360/platform/internal/data/contradiction/runtime"
)

var commonEntityKeys = []string{"customerid", "employeeid", "productid", "email", "accountnumber", "userid", "id"}

type EntityLinker struct{}

func NewEntityLinker() *EntityLinker {
	return &EntityLinker{}
}

func (l *EntityLinker) Link(models []*model.DataModel, sources map[string]*model.DataSource) []cruntime.ModelPair {
	pairs := make([]cruntime.ModelPair, 0)
	for i := 0; i < len(models); i++ {
		for j := i + 1; j < len(models); j++ {
			a := models[i]
			b := models[j]
			if a.SourceID == nil || b.SourceID == nil || *a.SourceID == *b.SourceID {
				continue
			}
			sourceA := sources[a.SourceID.String()]
			sourceB := sources[b.SourceID.String()]
			if sourceA == nil || sourceB == nil {
				continue
			}
			if link := findLinkColumn(a, b); link != "" {
				pairs = append(pairs, cruntime.ModelPair{
					ModelA:     a,
					ModelB:     b,
					SourceA:    sourceA,
					SourceB:    sourceB,
					LinkColumn: link,
				})
			}
		}
	}
	return pairs
}

func findLinkColumn(a, b *model.DataModel) string {
	fieldsA := make(map[string]model.ModelField, len(a.SchemaDefinition))
	for _, field := range a.SchemaDefinition {
		fieldsA[normalize(field.Name)] = field
	}
	for _, preferred := range commonEntityKeys {
		fieldA, okA := fieldsA[preferred]
		if !okA {
			continue
		}
		for _, fieldB := range b.SchemaDefinition {
			if normalize(fieldB.Name) == preferred && strings.EqualFold(fieldA.DataType, fieldB.DataType) {
				return fieldA.Name
			}
		}
	}
	for _, fieldA := range a.SchemaDefinition {
		name := normalize(fieldA.Name)
		for _, fieldB := range b.SchemaDefinition {
			if normalize(fieldB.Name) == name && strings.EqualFold(fieldA.DataType, fieldB.DataType) {
				return fieldA.Name
			}
		}
	}
	return ""
}

func normalize(value string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(value)), "_", "")
}
