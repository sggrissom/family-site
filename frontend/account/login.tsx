import * as preact from "preact"
import { Footer, Header } from "home";
import * as vlens from "vlens";
import * as server from "@app/server";

type Form = {
    email: string
    password: string
    error: string
}

const useForm = vlens.declareHook((): Form => ({
    email:"", password: "", error: ""
}))

export async function fetch(route: string, prefix: string) {
    return vlens.rpcOk({})
}

export function view(route: string, prefix: string, data: server.LoginResponse): preact.ComponentChild {
    let form = useForm()
    return <>
        <Header />
        <div className={"container"}>
            <LoginForm form={form}/>
        </div>
        <Footer />
    </>
}

async function onLoginClicked(form: Form, event: Event) {
    event.preventDefault()
    let [resp, err] = await server.AuthUser({
        Email: form.email,
        Password: form.password,
    })
    if (!resp) {
        form.error = err
    }
    vlens.scheduleRedraw()
}

const LoginForm = ({form}: {form: Form}) => {
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
