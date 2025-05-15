import * as preact from "preact"
import { Footer, Header } from "home";
import * as vlens from "vlens";
import * as server from "@app/server";
import * as rpc from "vlens/rpc";
import * as core from "vlens/core";
import * as auth from "util/authCache";

type Form = {
    email: string
    password: string
    error: string
}

const useForm = vlens.declareHook((): Form => ({
    email:"", password: "", error: ""
}))

export async function fetch(route: string, prefix: string) {
    return server.GetAuthContext({})
}

export function view(route: string, prefix: string, data: server.AuthResponse): preact.ComponentChild {
    if (data.Id > 0) {
        auth.setAuth(data)
        core.setRoute('/')
    }
    let form = useForm()
    return <>
        <Header/>
        <div className={"container"}>
            <LoginForm form={form}/>
        </div>
        <Footer />
    </>
}

const nativeFetch = window.fetch.bind(window);
async function onLoginClicked(form: Form, event: Event) {
    event.preventDefault()

    const res = await nativeFetch('/api/login', {
        method: 'POST',
        headers: {
          'Content-Type': ' application/json'
        },
        body: JSON.stringify(form)
      });

    if (!res.ok) {
        form.error = "login failure"
    }

    const result = await res.json()
    if (result.Success) {
        rpc.setAuthHeaders({'x-auth-token': result.Token})
        auth.setAuth(result.Auth)
    }

    vlens.scheduleRedraw()
}

type LoginFormProps = { form: Form }
const LoginForm : preact.FunctionalComponent<LoginFormProps> = ({form}) => {
    return (
        <div>
            <h2>Register</h2>
            {form.error ?? form.error}
            <form onSubmit={vlens.cachePartial(onLoginClicked, form)} >
                <label htmlFor="email">Email:</label>
                <input
                    type="email" 
                    id="email"
                    {...vlens.attrsBindInput(vlens.ref(form, "email"))}
                    required
                />
                <br />

                <label htmlFor="password">Password:</label>
                <input
                    type="password" 
                    id="password"
                    {...vlens.attrsBindInput(vlens.ref(form, "password"))}
                    required
                />
                <br />

                <button onClick={vlens.cachePartial(onLoginClicked, form)}>Login</button>
            </form>
            <a href="/register" style={{ fontSize: 'small' }}>Register</a>
            <br/>
            <a id="forgotLink" href="/forgot" style={{ fontSize: 'small' }}>Forgot Password?</a>
            <div style={{ marginTop: '15px' }}>
                <a href="/login/google" style={{ fontSize: 'small' }}>Use Google Account</a>
            </div>
        </div>
    );
}
