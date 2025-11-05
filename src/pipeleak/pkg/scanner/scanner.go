package scanner

import (
"github.com/CompassSecurity/pipeleak/pkg/scanner/artifact"
"github.com/CompassSecurity/pipeleak/pkg/scanner/engine"
"github.com/CompassSecurity/pipeleak/pkg/scanner/rules"
"github.com/CompassSecurity/pipeleak/pkg/scanner/types"
)

type Finding = types.Finding
type PatternElement = types.PatternElement
type PatternPattern = types.PatternPattern
type SecretsPatterns = types.SecretsPatterns
type DetectionResult = types.DetectionResult

var InitRules = rules.InitRules
var DownloadRules = rules.DownloadRules
var AppendPipeleakRules = rules.AppendPipeleakRules

var DetectHits = engine.DetectHits

var DetectFileHits = artifact.DetectFileHits
var HandleArchiveArtifact = artifact.HandleArchiveArtifact
