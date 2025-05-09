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
    user: User
    isAdmin: boolean
}

export interface User {
    Id: number
    Email: string
    Creation: string
    LastLogin: string
    FirstName: string
    LastName: string
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

