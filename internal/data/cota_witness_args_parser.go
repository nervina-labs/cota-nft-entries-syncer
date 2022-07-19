package data

import (
	"context"

	"github.com/nervina-labs/cota-nft-entries-syncer/internal/biz"
	"github.com/nervina-labs/cota-nft-entries-syncer/internal/data/blockchain"
	ckbTypes "github.com/nervosnetwork/ckb-sdk-go/types"
)

type CotaWitnessArgsParser struct {
	client *CkbNodeClient
}

func NewCotaWitnessArgsParser(client *CkbNodeClient) CotaWitnessArgsParser {
	return CotaWitnessArgsParser{
		client: client,
	}
}

type cotaCell struct {
	output     *ckbTypes.CellOutput
	index      int
	outputData []byte
}

func (c CotaWitnessArgsParser) Parse(tx *ckbTypes.Transaction, txIndex uint32, cotaType SystemScript) ([]biz.Entry, error) {
	if !c.hasCotaCell(tx.Outputs, cotaType) {
		return nil, nil
	}
	return c.cotaEntries(tx, txIndex, cotaType)
}

func (c CotaWitnessArgsParser) isCotaCell(output *ckbTypes.CellOutput, cotaType SystemScript) bool {
	if output.Type == nil {
		return false
	}
	return output.Type.CodeHash == cotaType.CodeHash && output.Type.HashType == cotaType.HashType
}

// There are not cota cells in inputs for registry, otherwise the amount of cota cells in inputs and outputs must be same
func (c CotaWitnessArgsParser) cotaEntries(tx *ckbTypes.Transaction, txIndex uint32, cotaType SystemScript) ([]biz.Entry, error) {
	inputCotaCellGroups, err := c.inputCotaCellGroups(tx.Inputs, cotaType)
	if err != nil {
		return nil, err
	}
	outputCotaCellGroups, err := c.outputCotaCellGroups(tx.Outputs, tx.OutputsData, cotaType)
	if err != nil {
		return nil, err
	}
	cotaCells := make([]cotaCell, len(inputCotaCellGroups))
	var cotaCellsIndex int
	for typeHash, inputCotas := range inputCotaCellGroups {
		outputGroupCotaCells := outputCotaCellGroups[typeHash]
		firstCotaAtOutputGroup := outputGroupCotaCells[0]
		firstCotaAtInputGroup := inputCotas[0]

		cotaCells[cotaCellsIndex] = cotaCell{
			output:     firstCotaAtOutputGroup.output,
			index:      firstCotaAtInputGroup.index,
			outputData: firstCotaAtOutputGroup.outputData,
		}

		cotaCellsIndex++
	}

	var entries []biz.Entry
	for _, cotaCell := range cotaCells {
		witness := tx.Witnesses[cotaCell.index]
		if len(witness) == 0 {
			continue
		}
		witnessArgs := blockchain.WitnessArgsFromSliceUnchecked(witness)
		if witnessArgs.OutputType().IsSome() {
			outputType, err := witnessArgs.OutputType().IntoBytes()
			if err != nil {
				return nil, err
			}
			entries = append(entries, biz.Entry{
				OutputType: outputType.RawData(),
				LockScript: cotaCell.output.Lock,
				TxIndex:    txIndex,
				Version:    cotaCell.outputData[0],
				TxHash:     tx.Hash,
			})
		}
		if witnessArgs.InputType().IsSome() {
			inputType, err := witnessArgs.InputType().IntoBytes()
			if err != nil {
				return nil, err
			}
			entries = append(entries, biz.Entry{
				InputType:  inputType.RawData(),
				LockScript: cotaCell.output.Lock,
				TxIndex:    txIndex,
				Version:    cotaCell.outputData[0],
				TxHash:     tx.Hash,
			})
		}
	}
	return entries, nil
}

func (c CotaWitnessArgsParser) inputCotaCellGroups(inputs []*ckbTypes.CellInput, cotaType SystemScript) (map[string][]cotaCell, error) {
	cotaCells, err := c.inputCotaCells(inputs, cotaType)
	if err != nil {
		return nil, err
	}

	group := make(map[string][]cotaCell)
	for _, cell := range cotaCells {
		typeHash, err := cell.output.Type.Hash()
		if err != nil {
			return group, err
		}

		group[typeHash.String()] = append(group[typeHash.String()], cell)
	}

	return group, nil
}

func (c CotaWitnessArgsParser) hasCotaCell(outputs []*ckbTypes.CellOutput, cotaType SystemScript) (result bool) {
	for _, output := range outputs {
		if result = c.isCotaCell(output, cotaType); result {
			break
		}
	}
	return result
}

func (c CotaWitnessArgsParser) inputCotaCells(inputs []*ckbTypes.CellInput, cotaType SystemScript) ([]cotaCell, error) {
	var cotaCells []cotaCell
	for i := 0; i < len(inputs); i++ {
		prevOutpoint := inputs[i].PreviousOutput
		prevTx, err := c.client.Rpc.GetTransaction(context.TODO(), prevOutpoint.TxHash)
		if err != nil {
			return nil, err
		}
		prevCellOutput := prevTx.Transaction.Outputs[prevOutpoint.Index]
		if c.isCotaCell(prevCellOutput, cotaType) {
			cotaCells = append(cotaCells, cotaCell{
				output: prevCellOutput,
				index:  i,
			})
		}
	}
	return cotaCells, nil
}

func (c CotaWitnessArgsParser) outputCotaCells(outputs []*ckbTypes.CellOutput, outputsData [][]byte, cotaType SystemScript) ([]cotaCell, error) {
	var cotaCells []cotaCell
	for i := 0; i < len(outputs); i++ {
		if c.isCotaCell(outputs[i], cotaType) {
			cotaCells = append(cotaCells, cotaCell{
				output:     outputs[i],
				index:      i,
				outputData: outputsData[i],
			})
		}
	}
	return cotaCells, nil
}

func (c CotaWitnessArgsParser) outputCotaCellGroups(outputs []*ckbTypes.CellOutput, outputsData [][]byte, cotaType SystemScript) (map[string][]cotaCell, error) {
	cotaCells, err := c.outputCotaCells(outputs, outputsData, cotaType)
	if err != nil {
		return nil, err
	}

	group := make(map[string][]cotaCell)
	for _, cell := range cotaCells {
		typeHash, err := cell.output.Type.Hash()
		if err != nil {
			return group, err
		}

		group[typeHash.String()] = append(group[typeHash.String()], cell)
	}

	return group, nil
}
