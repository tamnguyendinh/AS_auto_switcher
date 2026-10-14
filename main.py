"""
Main entry point for Antigravity Multi-Account Switcher.
"""

import sys
from ui.app_window import run_app

if __name__ == "__main__":
    try:
        run_app()
    except Exception as e:
        print(f"Error starting application: {e}", file=sys.stderr)
        sys.exit(1)
