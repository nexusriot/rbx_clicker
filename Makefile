GOOS   := windows
GOARCH := amd64
EXT    := .exe

OUTDIR := dist

.PHONY: all clean

all: $(OUTDIR)/ckicker$(EXT) $(OUTDIR)/clicker3$(EXT)

$(OUTDIR):
	mkdir -p $(OUTDIR)

$(OUTDIR)/ckicker$(EXT): ckicker.go | $(OUTDIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $@ $<

$(OUTDIR)/clicker3$(EXT): clicker3.go | $(OUTDIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $@ $<

clean:
	rm -rf $(OUTDIR)
