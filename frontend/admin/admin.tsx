import * as preact from "preact"
import { Footer, Header } from "home"
import * as server from "@app/server"
import * as rpc from "vlens/rpc"
import * as core from "vlens/core"
import { getAuth } from "util/authCache"

export async function fetch(route: string, prefix: string) {
    return rpc.ok<server.Empty>({})
}

export function view(route: string, prefix: string, data: server.Empty): preact.ComponentChild {
    const auth = getAuth()
    if (!(auth && auth.IsAdmin)) {
        core.setRoute("/")
    }
    return <>
        <AdminHeader/>
        <div className="container family-dashboard">
            <h2>Admin Dashboard</h2>
            <p>admin stuff.</p>
        </div>
        <AdminFooter/>
    </>
}

const AdminHeader = () => {
    return (
        <header>
            <div className="logo">Family Site</div>
            <nav>
                <a href="/">Main Site</a>
            </nav>
        </header>
    );
};

const AdminFooter = () => {
    return (
        <>
            <footer>
                &copy; 2024 Family Site
            </footer>
            <core.debugVarsPanel />
        </>
    )
};
