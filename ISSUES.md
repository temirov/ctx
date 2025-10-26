# ISSUES (Append-only Log)

Entries record newly discovered requests or changes, with their outcomes. No instructive content lives here. Read @NOTES.md for the process to follow when fixing issues.

## Features

## Improvements

- [ ] [CT-02] add subcommands --copy-only, and add an abbreviated version of it, --co, which doesnt print the output of the command to STDOUT and only copies it to clipboard
- [ ] [CT-03] abbreviate --copy subcommand to --c, both staying valid and copying to the clipboard the output


## BugFixes

- [x] [CT-01] ctx doesnt respect excluded folders recursively

    the folder structure is like that
    13:29:14 tyemirov@computercat:~/Development/Research/PhotoCurator [master] $ tree -L 2
    .
    ├── art_critic
    │   ├── representatives.csv
    │   ├── results.csv
    │   ├── selected.csv
    │   ├── summary.txt
    │   └── winners_final
    ├── cluster_and_pick.py
    ├── curator_core.py
    ├── Dockerfile
    ├── geometry
    │   ├── representatives.csv
    │   ├── results.csv
    │   ├── selected.csv
    │   ├── summary.txt
    │   └── winners_final
    ├── ISSUES.md
    ├── photo_curator_ddd
    │   ├── curator
    │   ├── domain
    │   ├── server.py
    │   ├── site_data
    │   └── webui
    ├── __pycache__
    │   └── curator_core.cpython-311.pyc
    ├── README.md
    ├── representatives.csv
    ├── results.csv
    ├── selected.csv
    ├── server.py
    ├── site_data
    │   ├── jobs
    │   ├── runs
    │   └── sessions
    ├── summary.txt
    ├── top_best.py
    ├── webui
    │   └── index.html
    └── winners_final
        ├── 0001_20251022_135350.heic_compressed.JPEG -> /home/tyemirov/Downloads/download(1)/20251022_135350.heic_compressed.JPEG
        ├── 0002_20251022_135314.heic_compressed.JPEG -> /home/tyemirov/Downloads/download(1)/20251022_135314.heic_compressed.JPEG
        ├── 0003_20251022_135438.heic_compressed.JPEG -> /home/tyemirov/Downloads/download(1)/20251022_135438.heic_compressed.JPEG
        ├── 0004_20251022_135308.heic_compressed.JPEG -> /home/tyemirov/Downloads/download(1)/20251022_135308.heic_compressed.JPEG
        └── 0005_20251022_132329.heic_compressed.JPEG -> /home/tyemirov/Downloads/download(1)/20251022_132329.heic_compressed.JPEG

    pointing ctx to photo_curator_ddd folder I dont extect to see the content of the site_data folder because it is excluded in .gitignore
    13:30:55 tyemirov@computercat:~/Development/Research/PhotoCurator [master] $ ctx c .gitignore 
    {
    "path": "/home/tyemirov/Development/Research/PhotoCurator/.gitignore",
    "name": ".gitignore",
    "type": "file",
    "size": "81b",
    "lastModified": "2025-10-25 11:43",
    "mimeType": "text/plain; charset=utf-8",
    "content": "__pycache__/\nsite_data/\n\n*.csv\n*.png\n*.jpg\n*.jpeg\n*.JPG\n*.JPEG\n\nsummary.txt\n*.log"
    }
    and yet I can see all of the files there
    13:43:45 tyemirov@computercat:~/Development/Research/PhotoCurator/photo_curator_ddd [master] $ ctx t | grep site_data
        "path": "/home/tyemirov/Development/Research/PhotoCurator/photo_curator_ddd/site_data",
        "name": "site_data",
        "path": "/home/tyemirov/Development/Research/PhotoCurator/photo_curator_ddd/site_data/runs",
            "path": "/home/tyemirov/Development/Research/PhotoCurator/photo_curator_ddd/site_data/runs/088be44b9d",
            "path": "/home/tyemirov/Development/Research/PhotoCurator/photo_curator_ddd/site_data/runs/088be44b9d/results",
                "path": "/home/tyemirov/Development/Research/PhotoCurator/photo_curator_ddd/site_data/runs/088be44b9d/results/representatives.csv",
                "path": "/home/tyemirov/Development/Research/PhotoCurator/photo_curator_ddd/site_data/runs/088be44b9d/results/results.csv",
    - Resolved by applying ancestor ignore directives when walking nested directories; regression coverage added for parent .gitignore handling.

## Maintenance

## 
