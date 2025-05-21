import * as rpc from "vlens/rpc"

export interface AddFamilyRequest {
    Name: string
}

export interface FamilyListResponse {
    AllFamilyNames: string[]
}

export interface Empty {
}

export interface AddUserRequest {
    Email: string
    Password: string
    FirstName: string
    LastName: string
}

export interface UserListResponse {
}

export interface AuthResponse {
    Id: number
    Email: string
    FirstName: string
    LastName: string
    IsAdmin: boolean
}

export async function AddFamily(data: AddFamilyRequest): Promise<rpc.Response<FamilyListResponse>> {
    return await rpc.call<FamilyListResponse>('AddFamily', JSON.stringify(data));
}

export async function ListFamilies(data: Empty): Promise<rpc.Response<FamilyListResponse>> {
    return await rpc.call<FamilyListResponse>('ListFamilies', JSON.stringify(data));
}

export async function AddUser(data: AddUserRequest): Promise<rpc.Response<UserListResponse>> {
    return await rpc.call<UserListResponse>('AddUser', JSON.stringify(data));
}

export async function GetAuthContext(data: Empty): Promise<rpc.Response<AuthResponse>> {
    return await rpc.call<AuthResponse>('GetAuthContext', JSON.stringify(data));
}

