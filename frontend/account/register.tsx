import * as preact from "preact"
import { Footer, Header } from "home";
import * as vlens from "vlens";
import * as server from "@app/server";

type Form = {
    first: string
    last: string
    email: string
    password: string
    error: string
}

const useForm = vlens.declareHook((): Form => ({
    first: "", last: "", email:"", password: "", error: ""
}))

export async function fetch(route: string, prefix: string) {
    return vlens.rpcOk({})
}

export function view(route: string, prefix: string, data: server.UserListResponse): preact.ComponentChild {
    let form = useForm()
    return <>
        <Header />
        <div className={"container"}>
            <RegisterForm form={form}/>
        </div>
        <Footer />
    </>
}

async function onAddUserClicked(form: Form, event: Event) {
    event.preventDefault()
    let [resp, err] = await server.AddUser({
        Email: form.email,
        Password: form.password,
        FirstName: form.first,
        LastName: form.last,
    })
    if (resp) {
        form.first = ""
        form.last = ""
        form.password = ""
        form.email = ""
        form.error = ""
    } else {
        form.error = err
    }
    vlens.scheduleRedraw()
}

const RegisterForm = ({form}: {form: Form}) => {
    return (
        <div>
            <h2>Register</h2>
            {form.error ?? form.error}
            <form onSubmit={vlens.cachePartial(onAddUserClicked, form)} >
                <label htmlFor="first">First Name:</label>
                <input
                    type="text" 
                    id="first"
                    {...vlens.attrsBindInput(vlens.ref(form, "first"))}
                    required
                />
                <br />

                <label htmlFor="last">Last Name:</label>
                <input
                    type="text" 
                    id="last"
                    {...vlens.attrsBindInput(vlens.ref(form, "last"))}
                    required
                />
                <br />

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

                <input
                    type="text"
                    className="honeypot"
                    name="honeypot"
                    value={""}
                />
                <br />

                <button onClick={vlens.cachePartial(onAddUserClicked, form)}>Create Account</button>
            </form>
            <a href="/login">Already have an account?</a>

            <div style={{ marginTop: '15px' }}>
                <a href="/login/google" style={{ fontSize: 'small' }}>Use Google Account</a>
            </div>
        </div>
    );
}
