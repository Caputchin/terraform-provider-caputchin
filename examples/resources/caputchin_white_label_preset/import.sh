# Import id format: scope|id|game|axis|name
# scope is troop or site; game is empty for a widget-shell preset and may
# itself contain slashes for a game-axis preset (the pipe is the delimiter).

# Widget-shell (troop scope):
terraform import caputchin_white_label_preset.midnight "troop|troop_abc123||skin|midnight"

# Game-axis (troop scope, slashy game id):
terraform import caputchin_white_label_preset.leaf_hard "troop|troop_abc123|caputchin/games/leaf|configuration|hard"

# Per-site widget-shell:
terraform import caputchin_white_label_preset.blog_en "site|site_def456||locale|en"
