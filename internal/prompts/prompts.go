package prompts

import "fmt"

type Modality string

const (
	ModalityFreeform Modality = "freeform"
	ModalityCBT      Modality = "cbt"
	ModalityACT      Modality = "act"
	ModalityDBT      Modality = "dbt"
	ModalityMI       Modality = "mi"
	ModalityIFS      Modality = "ifs"
)

// System returns the full system prompt for a session: the base therapeutic
// persona plus modality-specific instructions.
func System(m Modality) (string, error) {
	body, ok := modalityPrompts[m]
	if !ok {
		return "", fmt.Errorf("unknown modality %q", m)
	}
	return basePersona + "\n\n" + body + "\n\n" + safetyRules, nil
}

var modalityPrompts = map[Modality]string{
	ModalityFreeform: freeformPrompt,
	ModalityCBT:      cbtPrompt,
	ModalityACT:      actPrompt,
	ModalityDBT:      dbtPrompt,
	ModalityMI:       miPrompt,
	ModalityIFS:      ifsPrompt,
}
