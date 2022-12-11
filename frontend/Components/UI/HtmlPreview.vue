<template>
    <div class="html-preview">
        <div class="html-preview__wrap">
            <button class="html-preview__btn" @click="device = 'mobile'" :class="{'active': device == 'mobile'}">
                <svg class="html-preview__btn-icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512.1 512.1">
                    <path d="M128 406c4 0 8-4 8-8V124a8 8 0 0 0-15 0v274c0 4 3 8 7 8zM384 264c-4 0-7 3-7 7v204c0 12-11 22-23 22H158c-12 0-22-10-22-22v-47a8 8 0 0 0-15 0v47c0 20 16 37 37 37h196c21 0 38-17 38-37V271c0-4-4-7-8-7zM354 0H158c-21 0-37 17-37 38v56a8 8 0 0 0 15 0V38c0-13 10-23 22-23h31c8 15 24 25 41 25h52c17 0 33-10 41-25h31c12 0 23 10 23 23v203a8 8 0 0 0 15 0V38c0-21-17-38-38-38zm-72 25h-52c-9 0-17-4-23-10h98c-6 6-14 10-23 10z"/>
                    <path d="M287 478a8 8 0 0 0 0-15h-62a8 8 0 0 0 0 15h62z"/>
                </svg>
            </button>
            <button @click="device = 'tablet'" class="html-preview__btn" :class="{'active': device == 'tablet'}">
                <svg class="html-preview__btn-icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512.1 512.1">
                    <path d="M408 0H104C80 0 60 20 60 44v26a8 8 0 0 0 15 0V44c0-16 13-29 29-29h304c16 0 29 13 29 29v424c0 16-13 29-29 29H104c-16 0-29-13-29-29V100a8 8 0 0 0-15 0v368c0 24 20 44 44 44h304c24 0 44-20 44-44V44c0-24-20-44-44-44z"/>
                    <path d="M90 61v284a8 8 0 0 0 15 0V69h302v103a8 8 0 0 0 15 0V61c0-4-3-7-7-7H97c-4 0-7 3-7 7zM90 432c0 4 3 8 7 8h318c4 0 7-4 7-8V202a8 8 0 0 0-15 0v223H105v-50a8 8 0 0 0-15 0v57zM233 467a23 23 0 1 0 46-1 23 23 0 0 0-46 1zm31 0a8 8 0 1 1-16 0 8 8 0 0 1 16 0zM288 42a8 8 0 0 0 0-15h-34a8 8 0 0 0 0 15h34z"/>
                    <circle cx="225.6" cy="34.3" r="7.5"/>
                </svg>
            </button>
            <button @click="device = 'desktop'" class="html-preview__btn" :class="{'active': device == 'desktop'}">
                <svg class="html-preview__btn-icon h-7" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 510.5 510.5">
                    <path d="M473 30H148a7 7 0 1 0 0 15h325c12 0 23 10 23 22v265h-50a8 8 0 0 0 0 15h50v31c0 13-11 23-23 23H117a7 7 0 1 0 0 15h70l-14 50h-25a7 7 0 1 0 0 15h215a7 7 0 1 0 0-15h-26l-13-50h149c21 0 38-17 38-38V67c0-20-17-37-38-37zM322 466H189l13-50h106l14 50z"/>
                    <path d="M87 401H38c-13 0-23-10-23-23v-31h401a7 7 0 1 0 0-15H15V67c0-12 10-22 23-22h80a7 7 0 1 0 0-15H38C17 30 0 47 0 67v311c0 21 17 38 38 38h49a7 7 0 1 0 0-15z"/>
                </svg>
            </button>
        </div>
        <div :class="`device-${device}`">
            <div>
                <slot></slot>
            </div>
        </div>
    </div>
</template>

<script>
export default {
    props: {
        device: {
            default() {
                return 'desktop'
            }
        }
    }

}
</script>
