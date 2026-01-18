# Erato package derivation
# https://ryantm.github.io/nixpkgs/languages-frameworks/go/
{
  lib,
  buildGoModule,
}:
buildGoModule {
  pname = "erato";
  version = "0.1.0";

  src = lib.fileset.toSource {
    root = ../.;
    fileset = lib.fileset.unions [
      ../go.mod
      ../go.sum
      ../cmd
      ../internal
    ];
  };

  # Run `nix build` once to get the correct hash, then update this value
  vendorHash = "sha256-OhVA+dj4+ohhY3OCHa/FIrbhwhDb4aOLSq715RXNRcY=";

  subPackages = [ "cmd/erato" ];

  # Pure Go build (SQLite uses modernc.org/sqlite which is pure Go)
  env.CGO_ENABLED = 0;

  ldflags = [
    "-s"
    "-w"
  ];

  meta = {
    description = "Archive proxy and tracking tool";
    homepage = "https://github.com/stolasapp/erato";
    license = lib.licenses.asl20;
    mainProgram = "erato";
  };
}
