import * as rpc from "vlens/rpc"

export interface AddFamilyRequest {
    Name: string
}

export interface FamilyListResponse {
    AllFamilyNames: string[]
}

export interface Empty {
}

export async function AddFamily(data: AddFamilyRequest): Promise<rpc.Response<FamilyListResponse>> {
    return await rpc.call<FamilyListResponse>('AddFamily', JSON.stringify(data));
}

export async function ListFamilies(data: Empty): Promise<rpc.Response<FamilyListResponse>> {
    return await rpc.call<FamilyListResponse>('ListFamilies', JSON.stringify(data));
}

