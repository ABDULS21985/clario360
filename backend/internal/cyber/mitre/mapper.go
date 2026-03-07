package mitre

import "github.com/clario360/platform/internal/cyber/model"

// MapRuleToPrimaryTechnique resolves the rule's first mapped ATT&CK technique and tactic.
func MapRuleToPrimaryTechnique(rule *model.DetectionRule) (*Technique, *Tactic) {
	if rule == nil || len(rule.MITRETechniqueIDs) == 0 {
		return nil, nil
	}
	technique, ok := TechniqueByID(rule.MITRETechniqueIDs[0])
	if !ok {
		return nil, nil
	}
	if len(technique.TacticIDs) == 0 {
		return technique, nil
	}
	tactic, _ := TacticByID(technique.TacticIDs[0])
	return technique, tactic
}
