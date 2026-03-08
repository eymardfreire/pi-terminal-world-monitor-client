"""
Shared color palette for severity, trend, and UI.
Use _style(term, attr, text) with these attribute names so all panels stay consistent.
"""
# Severity (alerts, status): critical -> elevated -> monitoring -> normal
SEVERITY = {
    "critical": "red",
    "elevated": "yellow",
    "monitoring": "cyan",
    "normal": "green",
}

# Trend (markets, deltas): up / down / neutral
TREND = {
    "up": "green",
    "down": "red",
    "neutral": "dim",
}

# UI roles
UI = {
    "app_title": "bold_cyan",
    "panel_title": "cyan",
    "border": "cyan",
    "footer": "dim",
    "placeholder": "yellow",
    "error": "red",
}
