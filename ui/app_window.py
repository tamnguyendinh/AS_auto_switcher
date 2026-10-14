"""
Modern GUI for Antigravity Multi-Account Switcher using Tkinter.
Features:
- Dark theme UI
- Real-time Chrome account sync
- Status dots: 🟢 Active, 🟢 Ready, 🟡 Unsaved
- 1-Click hot-swap between saved JSON accounts
- [💾 Lưu phiên Antigravity hiện tại] button
- Always-on-top toggle
"""

import tkinter as tk
from tkinter import messagebox
import datetime
from typing import Optional
from core.state_resolver import AccountManager, AccountInfo


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


class SwitcherApp:
    def __init__(self, root: tk.Tk):
        self.root = root
        self.root.title("⚡ Antigravity Multi-Account Switcher")
        self.root.geometry("500x540")
        self.root.minsize(440, 480)
        self.root.configure(bg=BG_DARK)

        self.manager = AccountManager()
        self.always_on_top_var = tk.BooleanVar(value=True)
        self.root.attributes("-topmost", True)

        self._build_ui()
        self.refresh_accounts()

    def _build_ui(self):
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
            text="● Đang kiểm tra Antigravity...",
            font=FONT_BOLD,
            fg=COLOR_GREEN,
            bg=BG_CARD
        )
        self.active_status_label.pack(anchor="w", pady=(4, 0))

        # Top Controls Bar
        top_ctrl = tk.Frame(self.root, bg=BG_DARK)
        top_ctrl.pack(fill=tk.X, padx=14, pady=4)

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
            text="Sẵn sàng",
            font=FONT_SMALL,
            fg=FG_MUTED,
            bg=BG_DARK,
            anchor="w",
            padx=14,
            pady=4
        )
        self.status_bar.pack(fill=tk.X, side=tk.BOTTOM)

    def _on_canvas_configure(self, event):
        self.canvas.itemconfig(self.canvas_window, width=event.width)

    def _toggle_always_on_top(self):
        self.root.attributes("-topmost", self.always_on_top_var.get())

    def refresh_accounts(self):
        """Re-scans Chrome profiles and accounts_store/ to render the updated account list."""
        for widget in self.scrollable_frame.winfo_children():
            widget.destroy()

        active_email, accounts = self.manager.resolve_accounts()

        if active_email:
            self.active_status_label.config(
                text=f"● Đang hoạt động: {active_email}",
                fg=COLOR_GREEN
            )
        else:
            self.active_status_label.config(
                text="● Chưa phát hiện phiên Antigravity",
                fg=COLOR_YELLOW
            )

        if not accounts:
            empty_lbl = tk.Label(
                self.scrollable_frame,
                text="Không tìm thấy tài khoản Google nào trên Chrome.",
                bg=BG_DARK,
                fg=FG_MUTED,
                font=FONT_MAIN,
                pady=20
            )
            empty_lbl.pack(fill=tk.X)
        else:
            for acc in accounts:
                self._render_account_row(acc)

        stored_count = len(self.manager.store.list_stored_emails())
        now_str = datetime.datetime.now().strftime("%H:%M:%S")
        self.status_bar.config(
            text=f"Đồng bộ lúc {now_str} • {stored_count} tài khoản trong kho JSON • {len(accounts)} trên Chrome"
        )

    def _render_account_row(self, acc: AccountInfo):
        row = tk.Frame(self.scrollable_frame, bg=BG_CARD, padx=12, pady=10)
        row.pack(fill=tk.X, pady=4)

        left = tk.Frame(row, bg=BG_CARD)
        left.pack(side=tk.LEFT, fill=tk.X, expand=True)

        dot_color = COLOR_GREEN if acc.status_color == "green" else COLOR_YELLOW
        dot = tk.Label(left, text="●", font=FONT_BOLD, fg=dot_color, bg=BG_CARD)
        dot.pack(side=tk.LEFT, padx=(0, 6))

        email_lbl = tk.Label(
            left,
            text=acc.email,
            font=FONT_BOLD if acc.is_active else FONT_MAIN,
            fg="#ffffff" if acc.is_active else FG_TEXT,
            bg=BG_CARD
        )
        email_lbl.pack(side=tk.LEFT)

        # Action Buttons
        if acc.is_active:
            badge = tk.Label(
                row,
                text="ĐANG DÙNG",
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
        else: # UNAUTH (Not yet in accounts_store/)
            hint_lbl = tk.Label(
                row,
                text="Chưa lưu JSON",
                font=FONT_SMALL,
                fg=COLOR_YELLOW,
                bg=BG_CARD,
                padx=6
            )
            hint_lbl.pack(side=tk.RIGHT)

    def _on_switch_click(self, target_email: str):
        success, msg = self.manager.switch_account(target_email)
        if success:
            self.refresh_accounts()
            messagebox.showinfo(
                "Chuyển Đổi Thành Công",
                f"{msg}\n\nAntigravity đã nhận phiên mới và sẵn sàng tiếp tục công việc!"
            )
        else:
            messagebox.showerror("Lỗi Chuyển Đổi", msg)

    def _on_save_current_session(self):
        email = self.manager.auto_capture_active_session()
        if email:
            self.refresh_accounts()
            messagebox.showinfo(
                "Đã Lưu Phiên",
                f"Đã lưu thành công token của {email} vào thư mục accounts_store/{email}.json!\n\n"
                f"Tài khoản này đã sẵn sàng để tái sử dụng vĩnh viễn."
            )
        else:
            messagebox.showwarning(
                "Không Tìm Thấy Phiên",
                "Chưa phát hiện phiên đăng nhập Antigravity nào để lưu."
            )


def run_app():
    root = tk.Tk()
    app = SwitcherApp(root)
    root.mainloop()


if __name__ == "__main__":
    run_app()
