sed -i '/type diseaseData struct {/,/}/d' internal/systems/quarantine.go
sed -i '/type QuarantineSystem struct {/a \	activeDiseases []diseaseData' internal/systems/quarantine.go
sed -i '/toAdd:       make(\[\]ecs.Entity, 0, 50),/a \		activeDiseases: make([]diseaseData, 0, 100),' internal/systems/quarantine.go

sed -i '/var activeDiseases \[\]diseaseData/d' internal/systems/quarantine.go
sed -i 's/activeDiseases = append(activeDiseases/s.activeDiseases = append(s.activeDiseases/g' internal/systems/quarantine.go
sed -i 's/len(activeDiseases)/len(s.activeDiseases)/g' internal/systems/quarantine.go
sed -i 's/d := &activeDiseases\[i\]/d := \&s.activeDiseases[i]/g' internal/systems/quarantine.go
sed -i '/diseaseFilter := ecs.All(diseasePosID, diseaseEntID)/i \\ts.activeDiseases = s.activeDiseases[:0]' internal/systems/quarantine.go

sed -i '/import (/a \
type diseaseData struct {\n\tX float32\n\tY float32\n}\n' internal/systems/quarantine.go
