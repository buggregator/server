<template>
    <form v-if="isLoggedIn" :action="'/logout'" method="post" class="logout">
        <button type="submit" title="Logout" class="logout__btn-submit">
            <span>
                <LogoutIcon />
            </span>
        </button>
    </form>
</template>

<script>
import LogoutIcon from "@/Components/UI/Icons/LogoutIcon";

export default {
    components: {LogoutIcon},
    data() {
        return {
            user: null
        }
    },
    mounted() {
        this.user = this.$page.props.user.data || null
    },
    computed: {
        isLoggedIn() {
            return this.user !== null
        }
    }
}
</script>

<style lang="scss">
.logout {
  &__btn-submit {
    @apply block p-3 md:p-4 lg:p-5 hover:bg-blue-500 hover:text-white text-gray-500;

    > span > svg {
      @apply w-full h-full;
    }
  }
}
</style>
