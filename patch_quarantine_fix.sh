sed -i '5,8d' internal/systems/quarantine.go
sed -i '/)/a \
\ntype diseaseData struct {\n\tX float32\n\tY float32\n}' internal/systems/quarantine.go
