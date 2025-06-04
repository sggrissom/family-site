import * as preact from "preact"
import * as server from "@app/server";
import * as vlens from "vlens";
import * as events from "vlens/events";
import { Footer, Header } from "./home";

export async function fetch(route: string, prefix: string) {
    return server.ListFamilies({})
}

type Form = {
    data: server.FamilyListResponse
    name: string
    error: string
}

const useForm = vlens.declareHook((data: server.FamilyListResponse): Form => ({
    data, name: "", error: ""
}))

export function view(route: string, prefix: string, data: server.FamilyListResponse): preact.ComponentChild {
    let form = useForm(data)
    return <>
        <Header />
        <div className={"container"}>
            <Families form={form}/>
        </div>
        <Footer />
    </>
}

const Families = ({form}: {form: Form}) => {
    return <div>
        <h3>Families</h3>
        {form.data.AllFamilyNames.map(name => <div key={name}>{name}</div>)}
        <h3>Add Family</h3>
        <input type="text" {...events.inputAttrs(vlens.ref(form, "name"))} />
        {form.name && <div>
            You are inputting: <code>{form.name}</code>
        </div>}
        <button onClick={vlens.cachePartial(onAddFamilyClicked, form)}>Add</button>
    </div>
}

async function onAddFamilyClicked(form: Form) {
    let [resp, err] = await server.AddFamily({Id: 0, Name: form.name, Description: ""})
    if (resp) {
        form.name = ""
        form.data = resp
        form.error = ""
    } else {
        form.error = err
    }
    vlens.scheduleRedraw()
}
