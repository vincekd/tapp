import {
    trigger,
    query,
    group,
    animate,
    transition,
    style
} from "@angular/animations";

const ltr: any = [
    query(':enter, :leave', style({
        position: 'fixed',
        width: '100%',
        height: '100%',
        overflow: 'hidden'
    }), { optional: true }),
    query('#tweet-area ', style({ overflow: 'hidden' }), {optional: true}),
    group([
        query(':enter', [
            style({ transform: 'translateX(100%)' }),
            animate('0.5s ease-in-out', style({ transform: 'translateX(0%)' }))
        ], { optional: true }),
        query(':leave', [
            style({ transform: 'translateX(0%)' }),
            animate('0.5s ease-in-out', style({ transform: 'translateX(-100%)' })),
        ], { optional: true })
    ])
];
const rtl: any = [
    query(':enter, :leave', style({
        position: 'fixed',
        width: '100%',
        height: '100%',
    }), { optional: true }),
    query('#tweet-area ', style({ overflow: 'hidden' }), {optional: true}),
    group([
        query(':enter', [
            style({ transform: 'translateX(-100%)' }),
            animate('0.5s ease-in-out', style({ transform: 'translateX(0%)' })),
        ], { optional: true }),
        query(':leave', [
            style({ transform: 'translateX(0%)' }),
            animate('0.5s ease-in-out', style({ transform: 'translateX(100%)' }))
        ], { optional: true }),
    ])
];

export const routerTransition = trigger('routerTransition', [
    transition('latest => *', ltr),
    transition('best => search', ltr),
    transition('best => tweet-id', ltr),
    transition('search => tweet-id', ltr),
    transition('best => latest', rtl),
    transition('search => best', rtl),
    transition('search => latest', rtl),
    transition('tweet-id => *', rtl),
]);
