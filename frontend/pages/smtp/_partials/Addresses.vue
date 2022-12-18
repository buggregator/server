<template>
    <div class="flex flex-col divide-y">
       <div class="flex items-center space-x-2" v-for="user in addresses">
           <span class="text-white bg-gray-800 px-2" v-if="user.name">{{ user.name }}</span> <span>{{ user.email }}</span>
       </div>
    </div>
</template>

<script>
export default {
    props: {
        addresses: Array
    }
}
</script>
