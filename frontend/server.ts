import * as rpc from "vlens/rpc"

export type PersonType = number;
export const Parent: PersonType = 0;
export const Child: PersonType = 1;

export type GenderType = number;
export const Male: GenderType = 0;
export const Female: GenderType = 1;
export const Undisclosed: GenderType = 2;

// Errors
export const ErrLoginFailure = "LoginFailure";
export const ErrAuthFailure = "AuthFailure";
export const ErrEmailTaken = "EmailTaken";
export const ErrPasswordInvalid = "PasswordInvalid";

export interface AddUserRequest {
    Email: string
    Password: string
    FirstName: string
    LastName: string
}

export interface UserListResponse {
}

export interface Empty {
}

export interface AuthResponse {
    Id: number
    Email: string
    FirstName: string
    LastName: string
    IsAdmin: boolean
}

export interface AddPersonRequest {
    Id: number
    PersonType: number
    Gender: number
    Birthdate: string
    Name: string
}

export interface PersonListResponse {
    AllPersonNames: string[]
}

export interface AddFamilyRequest {
    Id: number
    Name: string
    Description: string
}

export interface FamilyListResponse {
    AllFamilyNames: string[]
}

export interface FamilyDataResponse {
    AuthUserId: number
    Family: Family
    Members: Person[]
}

export interface Family {
    Id: number
    Name: string
    Description: string
    CreatorId: number
}

export interface Person {
    Id: number
    FamilyId: number
    Type: PersonType
    Gender: GenderType
    Name: string
    Birthday: string
    Age: string
    ImageId: number
}

export async function AddUser(data: AddUserRequest): Promise<rpc.Response<UserListResponse>> {
    return await rpc.call<UserListResponse>('AddUser', JSON.stringify(data));
}

export async function GetAuthContext(data: Empty): Promise<rpc.Response<AuthResponse>> {
    return await rpc.call<AuthResponse>('GetAuthContext', JSON.stringify(data));
}

export async function AddPerson(data: AddPersonRequest): Promise<rpc.Response<Empty>> {
    return await rpc.call<Empty>('AddPerson', JSON.stringify(data));
}

export async function ListPeople(data: Empty): Promise<rpc.Response<PersonListResponse>> {
    return await rpc.call<PersonListResponse>('ListPeople', JSON.stringify(data));
}

export async function AddFamily(data: AddFamilyRequest): Promise<rpc.Response<FamilyListResponse>> {
    return await rpc.call<FamilyListResponse>('AddFamily', JSON.stringify(data));
}

export async function ListFamilies(data: Empty): Promise<rpc.Response<FamilyListResponse>> {
    return await rpc.call<FamilyListResponse>('ListFamilies', JSON.stringify(data));
}

export async function GetFamilyInfo(data: Empty): Promise<rpc.Response<FamilyDataResponse>> {
    return await rpc.call<FamilyDataResponse>('GetFamilyInfo', JSON.stringify(data));
}

