import * as preact from "preact";
import * as rpc from "vlens/rpc";
import * as core from "vlens/core";
import * as css from "vlens/css";
import * as vlens from "vlens";
import { AuthCache, clearAuth, getAuth } from "./util/authCache";
import { Empty } from "./server";

type Data = {};

export async function fetch(route: string, prefix: string) {
  return rpc.ok<Data>({});
}

export function view(
  route: string,
  prefix: string,
  data: Data,
): preact.ComponentChild {
  const auth = getAuth();
  if (auth && auth.Id > 0) {
    core.setRoute("/dashboard");
    return;
  }

  return (
    <>
      <Header />
      <HeroSection />
      <Footer />
    </>
  );
}

const HeroSection = () => {
  return (
    <div className="hero">
      <h1>Welcome to a Family Site</h1>
      <p style={{ marginBottom: "40px" }}>Track some family stuff.</p>
      <a className={callToActionButton} href="/register">
        Get Started
      </a>
      <a className={callToActionButton} href="/explore">
        Explore
      </a>
    </div>
  );
};

export const Header = () => {
  const auth = getAuth();
  if (auth && auth.Id > 0) {
    return <LoggedInHeader />;
  }
  return <LoggedOutHeader />;
};

const LoggedOutHeader = () => {
  return (
    <header>
      <div className={headerLogo}>Family Site</div>
      <nav>
        <a href="/explore">Explore</a>
        <a href="/login">Log In</a>
        <a href="/register">Sign Up</a>
      </nav>
    </header>
  );
};

const LoggedInHeader = () => {
  return (
    <header>
      <div className="logo">Family Site</div>
      <nav>
        <a href="/dashboard">Dashboard</a>
        <a href="/explore">Explore</a>
        <a href="/" onClick={onLogoutClicked}>
          Logout
        </a>
      </nav>
    </header>
  );
};

const nativeFetch = window.fetch.bind(window);
async function onLogoutClicked(event: Event) {
  event.preventDefault();

  await nativeFetch("/api/logout", {
    method: "POST",
    headers: {
      "Content-Type": " application/json",
    },
  });

  rpc.setAuthHeaders({});
  clearAuth();

  core.setRoute("/");

  vlens.scheduleRedraw();
}

export const Footer = () => {
  const auth = getAuth();
  return (
    <>
      <footer>
        &copy; 2024 Family Site
        {auth && FooterLinks(auth)}
      </footer>
      <core.debugVarsPanel />
    </>
  );
};

const FooterLinks = (auth: AuthCache) => {
  return (
    <div className="footer-links">
      <a href="/profile">Account {auth.Email}</a>

      {auth.IsAdmin && <a href="/admin">Admin Dashboard</a>}
    </div>
  );
};

css.rule("button,.button", {
  margin: "3px",
});

css.rule("body", {
  margin: 0,
  "font-family": "Arial, sans-serif",
  "background-color": "#f4f4f4",
  color: "#333",
});
css.rule("header", {
  "background-color": "#fff",
  "border-bottom": "1px solid #ccc",
  padding: "10px 20px",
  display: "flex",
  "justify-content": "space-between",
  "align-items": "center",
});
css.rule("header .logo", {
  "font-size": "1.5em",
  "font-weight": "bold",
});
css.rule("nav a", {
  "margin-left": "15px",
  "text-decoration": "none",
  color: "#333",
  "font-size": "1em",
});
css.rule(".container", {
  padding: "40px 20px",
  "text-align": "center",
});
css.rule(".hero", {
  color: "#fff",
  padding: "60px 20px",
  "text-align": "center",
  position: "relative",
  background: "linear-gradient(135deg, #00b894 30%, #b2f2bb 100%)",
  "min-height": "60vh",
});
css.rule(".hero h1", {
  "font-size": "2.5em",
  "margin-bottom": "10px",
});
css.rule(".hero p", {
  "font-size": "1.2em",
  "margin-bottom": "20px",
});
const callToActionButton = css.cls("cta-button", {
  "background-color": "#6c5ce7",
  color: "#fff",
  border: "none",
  padding: "15px 30px",
  "font-size": "1.1em",
  cursor: "pointer",
  "text-decoration": "none",
  "border-radius": "5px",
  transition: "background-color 0.3s ease",
  "box-shadow": "0 2px 4px rgba(0, 0, 0, 0.2)",
  margin: "5px",
});
css.rule(callToActionButton + ":hover", {
  "background-color": "#5848c2",
  transform: "translateY(-2px)",
  "box-shadow": "0 4px 6px rgba(0, 0, 0, 0.2)",
});
css.rule("footer", {
  "text-align": "center",
  padding: "20px",
  "font-size": "0.9em",
  color: "#777",
  background: "#fff",
  "border-top": "1px solid #ccc",
});
css.rule("header", {
  padding: "10px 20px",
  display: "flex",
  "justify-content": "space-between",
  "align-items": "center",
});
const headerLogo = css.cls("header-logo", {
  "font-size": "1.5em",
  "font-weight": "bold",
  color: "#6c5ce7",
});
css.rule("nav a", {
  "margin-left": "15px",
  "text-decoration": "none",
  color: "#333",
});

