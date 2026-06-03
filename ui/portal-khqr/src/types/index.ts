export interface PaymentConfig {
    merchantId: string;
    apiKey: string;
    sandbox: boolean;
}

export interface QRDesign {
    darkColor: string;
    lightColor: string;
}

export interface QRData {
    imageUrl: string;
    generateAt: string;
}

export interface SidebarItem {
    id: string;
    label: string;
    icon: string;
    active?: boolean;
}

export type ThemeMode = 'light' | 'dark'