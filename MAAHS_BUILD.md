# Building the Maahs Version of Oh-My-Posh

- Sync the `main` branch of cmaahs/oh-my-posh with the upstream

```zsh
# change to the cmaahs/oh-my-posh repository directory
git checkout main
git fetch; git status; git pull
git checkout maahs
git merge main
# resolve conflicts & add
git push origin maahs
```

## Build

```zsh
git checkout maahs
cd src
go build -o ~/tbin/oh-my-posh
```

```zsh
cdwt  # choose: OMP-1/gitlab-mr-segment
git merge maahs
cd src
go build -o ~/tbin/oh-my-posh
```


