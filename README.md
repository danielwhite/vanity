# vanity: static redirect generator for custom import paths

This tool generates static HTML files for Golang projects hosted under
custom import paths.

Documentation is at: https://whitehouse.id.au/vanity

# Example

To regenerate the index files that support this tool:

```
$ go get whitehouse.id.au/vanity
$ go list whitehouse.id.au/... | vanity -replace example.com=github.com/danielwhite -o .
```

The following directory structure would be created under the working
directory:

```
whitehouse.id.au/
  vanity/
    index.html
```
