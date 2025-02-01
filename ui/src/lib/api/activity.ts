import { BASE_URL, PostWithAuth } from "./auth"

export const ActivitySVGURI = `${BASE_URL}/activity/svg`

export type ActivityData = {
    Id: string
    Title: string
    Author: string
    ThumbnailUrl: string
    Url: string
}

export async function GetActivity(): Promise<ActivityData> {
    const res = await fetch(`${BASE_URL}/activity`)
    if (!res.ok) {
        throw new Error(await res.text())
    }
    return res.json();
}

export async function ClearActivity(token: string): Promise<void> {
    const res = await fetch(`${BASE_URL}/activity/clear`, PostWithAuth(token))
    if (!res.ok) {
        throw new Error(await res.text())
    }
}