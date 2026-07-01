#!/usr/bin/env bash
set -e
cd "$(dirname "$0")"

for f in diagrams/*.d2; do
  nix run nixpkgs#d2 -- "$f" "public/diagrams/$(basename ${f%.d2}.svg)"
done

/nix/store/hvmbr5s05b0kzalrdvk53k1nvms9fws7-python3-3.12.13/bin/python3.12 - << 'EOF'
import sys; sys.path.insert(0, '/nix/store/q83gayjkx4l2ycxhqw7d6k45zp003m79-python3.12-qrcode-8.2/lib/python3.12/site-packages')
import qrcode, qrcode.image.svg
qr = qrcode.QRCode(error_correction=qrcode.constants.ERROR_CORRECT_L, box_size=10, border=2)
qr.add_data('https://slides.manuelzapf.io/from-clusters-to-controlplanes')
qr.make(fit=True)
qr.make_image(image_factory=qrcode.image.svg.SvgPathImage).save('public/qr-slides.svg')
EOF

npm run build -- --base /from-clusters-to-controlplanes/
chmod -R a+r dist/

docker build --platform linux/amd64 -t docker.io/santode/slides:latest .
docker push docker.io/santode/slides:latest
kubectl apply -f k8s/deployment.yaml
kubectl rollout restart deployment/slides -n slides
