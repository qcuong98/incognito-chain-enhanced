package committeestate

import (
	"github.com/incognitochain/incognito-chain/instruction"
)

//BuildIncurredInstructions : BuildIncurredInstruction from instructions
func (engine BeaconCommitteeEngine) BuildIncurredInstructions(
	env *BeaconCommitteeStateEnvironment) (
	[][]string, error) {
	newB := engine.uncommittedBeaconCommitteeStateV1
	incurredInstructions := [][]string{}
	if len(env.BeaconInstructions) == 0 {
		return incurredInstructions, nil
	}
	var err error

	env.subtituteCandidates, err = newB.getSubtituteCandidates()
	if err != nil {
		return nil, err
	}
	env.validators, err = newB.getValidators()
	if err != nil {
		return nil, err
	}
	for _, inst := range env.BeaconInstructions {
		switch inst[0] {
		case instruction.UNSTAKE_ACTION:
			unstakeInstruction, err := instruction.ValidateAndImportUnstakeInstructionFromString(inst)
			if err != nil {
				Logger.log.Errorf("SKIP unstake instruction %+v, error %+v", inst, err)
				return nil, err
			}
			_, incurredInsFromUnstake, err :=
				newB.processUnstakeInstruction(unstakeInstruction, env, nil)
			if err != nil {
				return nil, NewCommitteeStateError(ErrBuildIncurredInstruction, err)
			}
			if incurredInsFromUnstake != nil {
				incurredInstructions = append(incurredInstructions, incurredInsFromUnstake...)
			}
		}
	}
	engine.AbortUncommittedBeaconState()
	return incurredInstructions, nil
}
