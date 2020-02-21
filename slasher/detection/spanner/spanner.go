package spanner

import (
	"context"
	"fmt"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

// DetectionKind defines an enum type that
// gives us information on the type of slashable offense
// found when analyzing validator min-max spans.
type DetectionKind int

const (
	NotFound DetectionKind = iota
	DoubleVote
	SurroundVote
)

// DetectionResult tells us the kind of slashable
// offense found from detecting on min-max spans +
// the slashable epoch for the offense.
type DetectionResult struct {
	Kind           DetectionKind
	SlashableEpoch uint64
}

// Spanner defines a struct which can detect slashable
// attestation offenses by tracking validator min-max
// spans from validators .
type Spanner interface {
	UpdateSpan(ctx context.Context, att *ethpb.IndexedAttestation) error
	SpansForValidatorByEpoch(ctx context.Context, valIdx uint64, epoch uint64) ([2]uint16, error)
	ValidatorSpansByEpoch(ctx context.Context, epoch uint64) error
	DetectSlashing(ctx context.Context, sourceEpoch uint64) (*DetectionResult, error)
	DeleteValidatorSpansByEpoch(ctx context.Context, validatorIdx uint64, epoch uint64) error
}

type SpanDetector struct {
	// Slice of epochs for valindex => min-max span.
	spans []map[uint64][2]uint16
}

func NewSpanDetector() *SpanDetector {
	return &SpanDetector{
		spans: make([]map[uint64][2]uint16, 256),
	}
}

func (s *SpanDetector) SpansForValidatorByEpoch(ctx context.Context, valIdx uint64, epoch uint64) ([2]uint16, error) {
	numSpans := uint64(len(s.spans))
	if span := s.spans[epoch%numSpans]; span != nil {
		if minMaxSpan, ok := span[valIdx]; ok {
			return minMaxSpan, nil
		}
		return [2]uint16{}, fmt.Errorf("validator index %d not found in span map", valIdx)
	}
	return [2]uint16{}, fmt.Errorf("no span found for epoch %d", epoch)
}

// UpdateSpan given an indexed attestation for all of its attesting indices.
func (s *SpanDetector) UpdateSpan(ctx context.Context, att *ethpb.IndexedAttestation) error {
	numSpans := uint64(len(s.spans))
	source := att.Data.Source.Epoch
	target := att.Data.Target.Epoch
	// Update spansForEpoch[valIdx] using the source/target data for
	// each validator in attesting indices.
	for i := 0; i < len(att.AttestingIndices); i++ {
		valIdx := att.AttestingIndices[i]

		// Update min spans.
		for epoch := source - 1; epoch > 0; epoch-- {
			val := uint16(target - (epoch))
			if sp := s.spans[epoch%numSpans]; sp == nil {
				s.spans[epoch%numSpans] = make(map[uint64][2]uint16)
			}
			minSpan := s.spans[epoch%numSpans][valIdx][0]
			maxSpan := s.spans[epoch%numSpans][valIdx][1]
			if minSpan == 0 || minSpan > val {
				s.spans[epoch%numSpans][valIdx] = [2]uint16{val, maxSpan}
			} else {
				break
			}
		}

		// Update max spans.
		distance := target - source
		for epoch := uint64(1); epoch < distance; epoch++ {
			val := uint16(distance - epoch)
			if sp := s.spans[source+epoch%numSpans]; sp == nil {
				s.spans[source+epoch%numSpans] = make(map[uint64][2]uint16)
			}
			minSpan := s.spans[source+epoch%numSpans][valIdx][0]
			maxSpan := s.spans[source+epoch%numSpans][valIdx][1]
			if maxSpan < val {
				s.spans[source+epoch%numSpans][valIdx] = [2]uint16{minSpan, val}
			} else {
				break
			}
		}
	}
	return nil
}

//  ValidatorSpansByEpoch --
// TODO(#4587): Complete.
func (s *SpanDetector) ValidatorSpansByEpoch(ctx context.Context, epoch uint64) error {
	return nil
}

//  DeleteValidatorSpansByEpoch --
// TODO(#4587): Complete.
func (s *SpanDetector) DeleteValidatorSpansByEpoch(ctx context.Context, validatorIdx uint64, epoch uint64) error {
	return nil
}
