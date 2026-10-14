"""
Modern GUI for Antigravity Multi-Account Switcher using Tkinter.
Features:
- Dark theme UI
- Real-time Chrome account sync
- Live RAM active identity & Language Server status
- Status dots: 🟢 Active, 🟢 Ready, 🟡 Unsaved
- 1-Click hot-swap between saved JSON accounts
- [💾 Lưu phiên hiện tại] button
- Always-on-top toggle
- Graceful headless mode and custom manager injection support
"""

import tkinter as tk
from tkinter import messagebox
import datetime
from typing import Optional, List, Dict, Any
from core.state_resolver import AccountManager, AccountInfo, SwitchResult
from core.api_connector import find_language_server_process


# Modern Dark Palette
BG_DARK = "#181825"
BG_CARD = "#1e1e2e"
BG_CARD_HOVER = "#262638"
FG_TEXT = "#cdd6f4"
FG_MUTED = "#a6adc8"
COLOR_GREEN = "#a6e3a1"
COLOR_YELLOW = "#f9e2af"
COLOR_BLUE = "#89b4fa"
COLOR_RED = "#f38ba8"
FONT_MAIN = ("Segoe UI", 10)
FONT_BOLD = ("Segoe UI", 10, "bold")
FONT_TITLE = ("Segoe UI", 12, "bold")
FONT_SMALL = ("Segoe UI", 8)


class AppWindow:
    def __init__(
        self,
        root: Optional[tk.Tk] = None,
        manager: Optional[AccountManager] = None,
        headless: bool = False,
        show_dialogs: bool = True
    ):
        self.manager = manager or AccountManager()
        self.show_dialogs = show_dialogs
        self.headless = headless or (root is None)
        self.root = root

        # Model / Controller state
        self.active_email: Optional[str] = None
        self.is_language_server_detected: bool = False
        self.active_status_text: str = "● Đang kiểm tra Antigravity..."
        self.active_status_color: str = COLOR_YELLOW
        self.status_bar_text: str = "Sẵn sàng"
        self.empty_message: str = ""
        self.rendered_accounts: List[AccountInfo] = []
        self.account_badges: Dict[str, str] = {}
        self.switch_buttons: Dict[str, Any] = {}
        self.last_switch_result: Optional[SwitchResult] = None

        self.always_on_top_var = None

        if self.root is not None and not self.headless:
            try:
                self.root.title("⚡ Antigravity Multi-Account Switcher")
                self.root.geometry("500x540")
                self.root.minsize(440, 480)
                self.root.configure(bg=BG_DARK)

                self.always_on_top_var = tk.BooleanVar(value=True)
                try:
                    self.root.attributes("-topmost", True)
                except Exception:
                    pass

                self._build_ui()
            except (tk.TclError, Exception):
                self.headless = True

        self.refresh_accounts()

    def _build_ui(self):
        if self.headless or self.root is None:
            return

        # 1. Header Frame
        header = tk.Frame(self.root, bg=BG_CARD, padx=16, pady=12)
        header.pack(fill=tk.X, padx=12, pady=(12, 6))

        title_label = tk.Label(
            header,
            text="⚡ Antigravity Multi-Account Switcher",
            font=FONT_TITLE,
            fg=COLOR_BLUE,
            bg=BG_CARD
        )
        title_label.pack(anchor="w")

        self.active_status_label = tk.Label(
            header,
            text=self.active_status_text,
            font=FONT_BOLD,
            fg=self.active_status_color,
            bg=BG_CARD
        )
        self.active_status_label.pack(anchor="w", pady=(4, 0))

        # Top Controls Bar
        top_ctrl = tk.Frame(self.root, bg=BG_DARK)
        top_ctrl.pack(fill=tk.X, padx=14, pady=4)

        if self.always_on_top_var is not None:
            top_check = tk.Checkbutton(
                top_ctrl,
                text="Ghim trên cùng (Always on Top)",
                variable=self.always_on_top_var,
                command=self._toggle_always_on_top,
                bg=BG_DARK,
                fg=FG_MUTED,
                selectcolor=BG_CARD,
                activebackground=BG_DARK,
                activeforeground=FG_TEXT,
                font=FONT_SMALL
            )
            top_check.pack(side=tk.LEFT)

        save_btn = tk.Button(
            top_ctrl,
            text="💾 Lưu phiên hiện tại",
            font=FONT_SMALL,
            command=self._on_save_current_session,
            bg="#2a3f5f",
            fg=COLOR_BLUE,
            activebackground="#36507a",
            activeforeground="#ffffff",
            relief=tk.FLAT,
            padx=8,
            pady=2,
            cursor="hand2"
        )
        save_btn.pack(side=tk.RIGHT, padx=(4, 0))

        refresh_btn = tk.Button(
            top_ctrl,
            text="🔄 Làm mới Chrome",
            font=FONT_SMALL,
            command=self.refresh_accounts,
            bg=BG_CARD,
            fg=FG_TEXT,
            activebackground=BG_CARD_HOVER,
            activeforeground=COLOR_BLUE,
            relief=tk.FLAT,
            padx=8,
            pady=2,
            cursor="hand2"
        )
        refresh_btn.pack(side=tk.RIGHT)

        # 2. Main Account List Frame (Scrollable)
        list_container = tk.Frame(self.root, bg=BG_DARK)
        list_container.pack(fill=tk.BOTH, expand=True, padx=12, pady=6)

        self.canvas = tk.Canvas(list_container, bg=BG_DARK, highlightthickness=0)
        self.scrollbar = tk.Scrollbar(list_container, orient=tk.VERTICAL, command=self.canvas.yview)
        self.scrollable_frame = tk.Frame(self.canvas, bg=BG_DARK)

        self.scrollable_frame.bind(
            "<Configure>",
            lambda e: self.canvas.configure(scrollregion=self.canvas.bbox("all"))
        )

        self.canvas_window = self.canvas.create_window((0, 0), window=self.scrollable_frame, anchor="nw")
        self.canvas.configure(xscrollcommand=None, yscrollcommand=self.scrollbar.set)

        self.canvas.bind('<Configure>', self._on_canvas_configure)
        self.canvas.pack(side=tk.LEFT, fill=tk.BOTH, expand=True)
        self.scrollbar.pack(side=tk.RIGHT, fill=tk.Y)

        # 3. Legend / Guide Frame
        legend = tk.Frame(self.root, bg=BG_CARD, padx=12, pady=8)
        legend.pack(fill=tk.X, padx=12, pady=(4, 6))

        l_text = "🟢 Sẵn sàng chuyển đổi ngầm   |   🟡 Chưa có file JSON trong kho"
        tk.Label(
            legend,
            text=l_text,
            font=FONT_SMALL,
            fg=FG_MUTED,
            bg=BG_CARD
        ).pack(anchor="center")

        # 4. Status Bar
        self.status_bar = tk.Label(
            self.root,
            text=self.status_bar_text,
            font=FONT_SMALL,
            fg=FG_MUTED,
            bg=BG_DARK,
            anchor="w",
            padx=14,
            pady=4
        )
        self.status_bar.pack(fill=tk.X, side=tk.BOTTOM)

    def _on_canvas_configure(self, event):
        if hasattr(self, "canvas") and hasattr(self, "canvas_window"):
            try:
                self.canvas.itemconfig(self.canvas_window, width=event.width)
            except Exception:
                pass

    def _toggle_always_on_top(self):
        if self.root and self.always_on_top_var:
            try:
                self.root.attributes("-topmost", self.always_on_top_var.get())
            except Exception:
                pass

    def refresh_accounts(self):
        """Re-scans Chrome profiles and accounts_store/ to render the updated account list."""
        if hasattr(self, "scrollable_frame"):
            for widget in self.scrollable_frame.winfo_children():
                try:
                    widget.destroy()
                except Exception:
                    pass

        self.switch_buttons.clear()
        self.account_badges.clear()

        # Query live state from AccountManager
        try:
            active_email, accounts = self.manager.resolve_accounts()
        except Exception:
            active_email, accounts = None, []

        self.active_email = active_email
        self.rendered_accounts = list(accounts)

        # Check if language server is detected
        ls_running = False
        try:
            finder = getattr(self.manager, "process_finder", None) or find_language_server_process
            ls_running = bool(finder())
        except Exception:
            ls_running = False
        self.is_language_server_detected = ls_running

        # Header status presentation
        if active_email:
            if ls_running:
                status_text = f"● Đang hoạt động: {active_email}"
            else:
                status_text = f"● Đang hoạt động: {active_email} (Credential fallback)"
            status_color = COLOR_GREEN
        else:
            status_text = "● Chưa phát hiện phiên Antigravity"
            status_color = COLOR_YELLOW

        self.active_status_text = status_text
        self.active_status_color = status_color

        if hasattr(self, "active_status_label"):
            try:
                self.active_status_label.config(text=status_text, fg=status_color)
            except Exception:
                pass

        # Render rows or empty notice
        if not accounts:
            self.empty_message = "Không tìm thấy tài khoản Google nào trên Chrome."
            if hasattr(self, "scrollable_frame"):
                try:
                    empty_lbl = tk.Label(
                        self.scrollable_frame,
                        text=self.empty_message,
                        bg=BG_DARK,
                        fg=FG_MUTED,
                        font=FONT_MAIN,
                        pady=20
                    )
                    empty_lbl.pack(fill=tk.X)
                except Exception:
                    pass
        else:
            self.empty_message = ""
            for acc in accounts:
                self._render_account_row(acc)

        # Status bar update
        stored_count = 0
        try:
            if hasattr(self.manager, "store") and hasattr(self.manager.store, "list_stored_emails"):
                stored_count = len(self.manager.store.list_stored_emails())
        except Exception:
            stored_count = 0

        now_str = datetime.datetime.now().strftime("%H:%M:%S")
        self.status_bar_text = f"Đồng bộ lúc {now_str} • {stored_count} tài khoản trong kho JSON • {len(accounts)} trên Chrome"
        if hasattr(self, "status_bar"):
            try:
                self.status_bar.config(text=self.status_bar_text)
            except Exception:
                pass

    def _render_account_row(self, acc: AccountInfo):
        self.account_badges[acc.email] = acc.display_badge

        if self.headless or not hasattr(self, "scrollable_frame"):
            return

        row = tk.Frame(self.scrollable_frame, bg=BG_CARD, padx=12, pady=10)
        row.pack(fill=tk.X, pady=4)

        left = tk.Frame(row, bg=BG_CARD)
        left.pack(side=tk.LEFT, fill=tk.X, expand=True)

        dot_color = COLOR_GREEN if acc.status_color == "green" else COLOR_YELLOW
        dot = tk.Label(left, text="●", font=FONT_BOLD, fg=dot_color, bg=BG_CARD)
        dot.pack(side=tk.LEFT, padx=(0, 8))

        # Text container for Name & Email
        text_frame = tk.Frame(left, bg=BG_CARD)
        text_frame.pack(side=tk.LEFT, fill=tk.X, expand=True)

        if acc.name:
            name_lbl = tk.Label(
                text_frame,
                text=acc.name,
                font=FONT_BOLD if acc.is_active else FONT_MAIN,
                fg="#ffffff" if acc.is_active else FG_TEXT,
                bg=BG_CARD,
                anchor="w"
            )
            name_lbl.pack(anchor="w")

            email_lbl = tk.Label(
                text_frame,
                text=acc.email,
                font=FONT_SMALL,
                fg=COLOR_GREEN if acc.is_active else FG_MUTED,
                bg=BG_CARD,
                anchor="w"
            )
            email_lbl.pack(anchor="w")
        else:
            email_lbl = tk.Label(
                text_frame,
                text=acc.email,
                font=FONT_BOLD if acc.is_active else FONT_MAIN,
                fg="#ffffff" if acc.is_active else FG_TEXT,
                bg=BG_CARD,
                anchor="w"
            )
            email_lbl.pack(anchor="w")

        # Action Badge / Button
        if acc.is_active:
            badge = tk.Label(
                row,
                text=acc.display_badge or "[ĐANG DÙNG]",
                font=FONT_SMALL,
                fg=COLOR_GREEN,
                bg="#1c3a27",
                padx=8,
                pady=4
            )
            badge.pack(side=tk.RIGHT)
        elif acc.status_code == "READY":
            switch_btn = tk.Button(
                row,
                text="⚡ Chuyển sang",
                font=FONT_MAIN,
                command=lambda em=acc.email: self._on_switch_click(em),
                bg="#2e5c38",
                fg="#ffffff",
                activebackground="#3b7648",
                activeforeground="#ffffff",
                relief=tk.FLAT,
                cursor="hand2",
                padx=10,
                pady=3
            )
            switch_btn.pack(side=tk.RIGHT)
            self.switch_buttons[acc.email] = switch_btn
        else: # UNAUTH (Not yet in accounts_store/)
            hint_lbl = tk.Label(
                row,
                text=acc.display_badge or "[Chưa lưu vào kho]",
                font=FONT_SMALL,
                fg=COLOR_YELLOW,
                bg=BG_CARD,
                padx=6
            )
            hint_lbl.pack(side=tk.RIGHT)

    def _on_switch_click(self, target_email: str) -> SwitchResult:
        # Provide immediate visual feedback
        if target_email in self.switch_buttons:
            btn = self.switch_buttons[target_email]
            if hasattr(btn, "config"):
                try:
                    btn.config(state=tk.DISABLED, text="Đang chuyển...")
                except Exception:
                    pass

        self.status_bar_text = f"Đang chuyển đổi sang {target_email}..."
        if hasattr(self, "status_bar"):
            try:
                self.status_bar.config(text=self.status_bar_text)
            except Exception:
                pass

        if self.root and hasattr(self.root, "update_idletasks"):
            try:
                self.root.update_idletasks()
            except Exception:
                pass

        # Execute switch via AccountManager
        res = self.manager.switch_account(target_email)
        self.last_switch_result = res
        success, msg = res

        # Refresh UI
        self.refresh_accounts()

        if success:
            now_str = datetime.datetime.now().strftime("%H:%M:%S")
            self.status_bar_text = f"Đã chuyển đổi thành công sang: {target_email} lúc {now_str}"
            if hasattr(self, "status_bar"):
                try:
                    self.status_bar.config(text=self.status_bar_text)
                except Exception:
                    pass
            if self.show_dialogs and not self.headless:
                try:
                    messagebox.showinfo(
                        "Chuyển Đổi Thành Công",
                        f"{msg}\n\nAntigravity đã nhận phiên mới và sẵn sàng tiếp tục công việc!"
                    )
                except Exception:
                    pass
        else:
            self.status_bar_text = f"Lỗi chuyển đổi: {msg}"
            if hasattr(self, "status_bar"):
                try:
                    self.status_bar.config(text=self.status_bar_text)
                except Exception:
                    pass
            if self.show_dialogs and not self.headless:
                try:
                    messagebox.showerror("Lỗi Chuyển Đổi", msg)
                except Exception:
                    pass

        return res

    def _on_save_current_session(self) -> Optional[str]:
        email = self.manager.auto_capture_active_session()
        if email:
            self.refresh_accounts()
            if self.show_dialogs and not self.headless:
                try:
                    messagebox.showinfo(
                        "Đã Lưu Phiên",
                        f"Đã lưu thành công token của {email} vào thư mục accounts_store/{email}.json!\n\n"
                        f"Tài khoản này đã sẵn sàng để tái sử dụng vĩnh viễn."
                    )
                except Exception:
                    pass
        else:
            if self.show_dialogs and not self.headless:
                try:
                    messagebox.showwarning(
                        "Không Tìm Thấy Phiên",
                        "Chưa phát hiện phiên đăng nhập Antigravity nào để lưu."
                    )
                except Exception:
                    pass
        return email


SwitcherApp = AppWindow


def run_app():
    try:
        root = tk.Tk()
        app = AppWindow(root)
        root.mainloop()
    except (tk.TclError, Exception) as e:
        print(f"Tkinter GUI initialization failed (headless or no display): {e}")
        app = AppWindow(headless=True)
        print("Initialized in headless fallback mode successfully.")
        return app


if __name__ == "__main__":
    run_app()
