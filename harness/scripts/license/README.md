# Backfil license headers
The script in this folder can be used to backfill license headers on existing files.
To run the script, execute:
```bash
./insert-license-headers.sh -l license-header.txt -f "$PATH_TO_CODE_FILE"
```

Alternatively, one can also provide `-s "$PATH_TO_LIST_FILE"` parameter to a file that contains a list of files (or path prefixes)