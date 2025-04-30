import * as preact from "preact"
import { Footer, Header } from "home";
import * as rpc from "vlens/rpc";

type Data = {}

export async function fetch(route: string, prefix: string) {
    return rpc.ok<Data>({})
}


export function view(route: string, prefix: string, data: Data): preact.ComponentChild {
    return <>
        <Header />
        <div className={"container"}>
            <RegisterForm/>
        </div>
        <Footer />
    </>
}

const handleSubmit = (e: Event) => {
    e.preventDefault();
};

const RegisterForm = () => {
    return (
        <div>
            <h2>Register</h2>
            <form onSubmit={handleSubmit}>
                <label htmlFor="firstname">First Name:</label>
                <input
                    type="text"
                    id="firstname"
                    name="firstname"
                    value={""}
                    required
                />
                <br />

                <label htmlFor="lastname">Last Name:</label>
                <input
                    type="text"
                    id="lastname"
                    name="lastname"
                    value={""}
                    required
                />
                <br />

                <label htmlFor="email">Email:</label>
                <input
                    type="email"
                    id="email"
                    name="email"
                    value={""}
                    required
                />
                <br />

                <label htmlFor="password">Password:</label>
                <input
                    type="password"
                    id="password"
                    value={""}
                    required
                />
                <br />

                <input
                    type="text"
                    className="honeypot"
                    name="honeypot"
                    value={""}
                />
                <br />

                <button type="submit">Create Account</button>
            </form>
            <a href="/login">Already have an account?</a>

            <div style={{ marginTop: '15px' }}>
                <a href="/login/google" style={{ fontSize: 'small' }}>Use Google Account</a>
            </div>
        </div>
    );
}
