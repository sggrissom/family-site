import * as preact from "preact";
import { Footer, Header } from "home";
import * as server from "@app/server";
import * as rpc from "vlens/rpc";
import * as core from "vlens/core";
import * as css from "vlens/css";
import { getAuth } from "util/authCache";

export async function fetch(route: string, prefix: string) {
  return rpc.ok<server.Empty>({});
}

const adminBody = css.cls("admin-body", {
  "background-color": "#ecf0f1",
  color: "#333",
  "line-height": "1.6",
});

css.rule("header", {
  "background-color": "#2c3e50",
  color: "#fff",
  padding: "1rem 2rem",
});

css.rule("header h1", {
  margin: 0,
});

css.rule("header nav a", {
  color: "#fff",
  "margin-left": "1rem",
  "text-decoration": "none",
});

css.rule("header nav a:hover", {
  "text-decoration": "underline",
});

const container = css.cls("admin_container", {
  display: "flex",
  "min-height": "calc(100vh - 140px)",
});

export function view(
  route: string,
  prefix: string,
  data: server.Empty,
): preact.ComponentChild {
  const auth = getAuth();
  if (!(auth && auth.IsAdmin)) {
    core.setRoute("/");
  }
  return preact.h("div", { class: adminBody }, [
    preact.h(AdminHeader),
    preact.h("div", { class: container }, [
      preact.h(AdminSidebar),
      preact.h("h2", {}, "Admin Dashboard"),
      preact.h("p", {}, "admin stuff."),
    ]),
    preact.h(AdminFooter),
  ]);
}

const headerContainer = css.cls("admin_header_container", {
  display: "flex",
  "justify-content": "space-between",
  "align-items": "center",
});

const headerTitle = css.cls("admin_header_container", {
  margin: "0",
});

const AdminHeader = () => {
  return preact.h(
    "header",
    {},
    preact.h("div", { class: headerContainer }, [
      preact.h("h1", { class: headerTitle }, "Admin Panel"),
      preact.h("nav", {}, [
        preact.h("a", { href: "/" }, "Home"),
        preact.h("a", { href: "/logout" }, "Logout"),
      ]),
    ]),
  );
};

const sidebar = css.cls("admin_sidebar", {
  width: "220px",
  "background-color": "#34495e",
  padding: "1rem",
});

const AdminSidebar = () => {
  return preact.h(
    "aside",
    { class: sidebar },
    preact.h("ul", {}, [
      preact.h("li", {}, preact.h("a", { href: "/admin" }, "Dashboard")),
      preact.h(
        "li",
        {},
        preact.h("a", { href: "/admin/users" }, "Manage Users"),
      ),
      preact.h(
        "li",
        {},
        preact.h("a", { href: "/admin/families" }, "Manage Families"),
      ),
      preact.h(
        "li",
        {},
        preact.h("a", { href: "/admin/people" }, "Manage People"),
      ),
    ]),
  );
};

const AdminFooter = () => {
  return (
    <>
      <footer>&copy; 2024 Family Site</footer>
      <core.debugVarsPanel />
    </>
  );
};
