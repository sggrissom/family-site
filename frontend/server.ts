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

export interface AuthRequest {
    Email: string
    Password: string
}

export interface LoginResponse {
    Success: boolean
    Token: string
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

export async function AuthUser(data: AuthRequest): Promise<rpc.Response<LoginResponse>> {
    return await rpc.call<LoginResponse>('AuthUser', JSON.stringify(data));
}

