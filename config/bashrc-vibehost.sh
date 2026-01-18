if [ -n "${PS1:-}" ] && command -v starship >/dev/null 2>&1; then
  export STARSHIP_CONFIG=/root/.config/starship.toml
  eval "$(starship init bash)"
fi
