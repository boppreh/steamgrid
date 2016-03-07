set -e

env GOOS=linux GOARCH=386 go build -v
mkdir -p steamgrid-linux/steamgrid
mv steamgrid steamgrid-linux/steamgrid
cp -r "overlays by category" steamgrid-linux/steamgrid

env GOOS=windows GOARCH=386 go build -v
mkdir -p steamgrid-windows/steamgrid
mv steamgrid.exe steamgrid-windows/steamgrid
cp -r "overlays by category" steamgrid-windows/steamgrid
