import * as preact from "preact"
import * as rpc from "vlens/rpc";
import * as server from "@app/server";
import * as core from "vlens/core";
import * as vlens from "vlens";
import { clearAuth, getAuth } from "./util/authCache";

type Data = {}

export async function fetch(route: string, prefix: string) {
    return rpc.ok<Data>({})
}

export function view(route: string, prefix: string, data: Data): preact.ComponentChild {
    const auth = getAuth()
    if (auth && auth.Id > 0) {
        core.setRoute('/dashboard')
    }
    
    return <>
        <Header /><HeroSection /><Footer />
    </>
}

const HeroSection = () => {
    return (
        <div className="hero">
            <h1>Welcome to a Family Site</h1>
            <p style={{ marginBottom: '40px' }}>Track some family stuff.</p>
            <a className="cta-button" href="/register">Get Started</a>
            <a className="cta-button" href="/explore">Explore</a>
        </div>
    );
};

export const Header = () => {
    const auth = getAuth()
    if (auth && auth.Id > 0) {
        return <LoggedInHeader />
    }
    return <LoggedOutHeader />
}

const LoggedOutHeader = () => {
    return (
        <header>
            <div className="logo">Family Site</div>
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
                <a onClick={onLogoutClicked}>Logout</a>
            </nav>
        </header>
    );
};

const nativeFetch = window.fetch.bind(window);
async function onLogoutClicked(event: Event) {
    event.preventDefault()

    await nativeFetch('/api/logout', {
        method: 'POST',
        headers: {
            'Content-Type': ' application/json'
        },
    });

    rpc.setAuthHeaders({})
    clearAuth()

    core.setRoute('/')

    vlens.scheduleRedraw()
}

export const Footer = () => {
    return (
        <footer>
            &copy; 2024 Family Site
        </footer>
    )
}