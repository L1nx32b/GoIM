<template>
  <div class="chat-wrap">
    <div
      class="chat-window"
      :style="{
        boxShadow: `var(${'--el-box-shadow-dark'})`,
      }"
    >
      <el-container class="chat-window-container">
        <el-aside class="aside-container">
          <NavigationModal></NavigationModal>
        </el-aside>
        <div class="owner-info-window">
          <div class="my-homepage-title"><h2>我的主页</h2></div>

          <p class="owner-prefix">用户id：{{ userInfo.uuid }}</p>
          <p class="owner-prefix">昵称：{{ userInfo.nickname }}</p>
          <p class="owner-prefix">电话：{{ userInfo.telephone }}</p>
          <p class="owner-prefix">邮箱：{{ userInfo.email }}</p>
          <p class="owner-prefix">
            性别：{{ userInfo.gender === 0 ? "男" : "女" }}
          </p>
          <p class="owner-prefix">生日：{{ userInfo.birthday }}</p>
          <p class="owner-prefix">个性签名：{{ userInfo.signature }}</p>
          <p class="owner-prefix">
            加入的时间：{{ userInfo.created_at }}
          </p>
          <div class="owner-opt">
            <p class="owner-prefix">头像：</p>
            <img style="width: 40px; height: 40px" :src="userInfo.avatar" />
          </div>
        </div>
        <div class="edit-window">
          <el-button class="edit-btn" @click="showMyInfoModal">编辑</el-button>
        </div>
        <Modal :isVisible="isMyInfoModalVisible">
          <template v-slot:header>
            <div class="modal-header">
              <div class="modal-quit-btn-container">
                <button class="modal-quit-btn" @click="quitMyInfoModal">
                  <el-icon><Close /></el-icon>
                </button>
              </div>
              <div class="modal-header-title">
                <h3>修改主页</h3>
              </div>
            </div>
          </template>
          <template v-slot:body>
            <el-scrollbar
              max-height="300px"
              style="
                width: 400px;
                display: flex;
                align-items: center;
                justify-content: center;
                margin-top: 20px;
              "
            >
              <div class="modal-body">
                <el-form ref="formRef" :model="updateInfo" label-width="70px">
                  <el-form-item
                    prop="nickname"
                    label="昵称"
                    :rules="[
                      {
                        min: 3,
                        max: 10,
                        message: '昵称长度在 3 到 10 个字符',
                        trigger: 'blur',
                      },
                    ]"
                  >
                    <el-input
                      v-model="updateInfo.nickname"
                      placeholder="选填"
                    />
                  </el-form-item>
                  <el-form-item prop="email" label="邮箱">
                    <el-input v-model="updateInfo.email" placeholder="选填" />
                  </el-form-item>
                  <el-form-item prop="birthday" label="生日">
                    <el-input
                      v-model="updateInfo.birthday"
                      placeholder="选填，格式为2024.1.1"
                    />
                  </el-form-item>
                  <el-form-item prop="signature" label="个性签名">
                    <el-input
                      v-model="updateInfo.signature"
                      placeholder="选填"
                    />
                  </el-form-item>
                  <el-form-item prop="avatar" label="头像">
                    <el-upload
                      v-model:file-list="fileList"
                      ref="uploadRef"
                      :auto-upload="false"
                      :action="uploadPath"
                      :on-success="handleUploadSuccess"
                      :before-upload="beforeFileUpload"
                    >
                      <template #trigger>
                        <el-button
                          style="background-color: #DAA520; border: 4px solid #8B7355; color: #FFF; font-family: 'Courier New', monospace; image-rendering: pixelated; box-shadow: 4px 4px 0px #8B7355;"
                          >上传图片</el-button
                        >
                      </template>
                    </el-upload>
                  </el-form-item>
                </el-form>
              </div>
            </el-scrollbar>
          </template>
          <template v-slot:footer>
            <div class="modal-footer">
              <el-button class="modal-close-btn" @click="closeMyInfoModal">
                完成
              </el-button>
            </div>
          </template>
        </Modal>
      </el-container>
    </div>
  </div>
</template>

<script>
import { reactive, toRefs, onMounted, ref } from "vue";
import { useStore } from "vuex";
import axios from "axios";
import { useRouter } from "vue-router";
import Modal from "@/components/Modal.vue";
import { checkEmailValid } from "@/assets/js/valid.js";
import { generateString } from "@/assets/js/random.js";
import SmallModal from "@/components/SmallModal.vue";
import NavigationModal from "@/components/NavigationModal.vue";
import ContactListModal from "@/components/ContactListModal.vue";
import { ElMessage } from "element-plus";
export default {
  name: "OwnInfo",
  components: {
    Modal,
    SmallModal,
    ContactListModal,
    NavigationModal,
  },
  setup() {
    const router = useRouter();
    const store = useStore();
    const data = reactive({
      userInfo: store.state.userInfo,
      updateInfo: {
        uuid: "",
        nickname: "",
        email: "",
        birthday: "",
        signature: "",
        avatar: "",
      },
      isMyInfoModalVisible: false,
      ownListReq: {
        owner_id: "",
      },
      uploadRef: null,
      uploadPath: store.state.backendUrl + "/message/uploadAvatar",
      fileList: [],
      cnt: 0,
    });
    const showMyInfoModal = () => {
      data.isMyInfoModalVisible = true;
    };
    const closeMyInfoModal = async () => {
      console.log(data.fileList);
      if (
        data.updateInfo.nickname == "" &&
        data.fileList.length == 0 &&
        data.updateInfo.email == "" &&
        data.updateInfo.birthday == "" &&
        data.updateInfo.signature == ""
      ) {
        ElMessage("请至少修改一项");
        return;
      }
      if (data.updateInfo.nickname != "") {
        if (
          data.updateInfo.nickname.length < 3 ||
          data.updateInfo.nickname.length > 10
        ) {
          return;
        }
      }
      if (data.updateInfo.email != "") {
        if (!checkEmailValid(data.updateInfo.email)) {
          ElMessage("请输入有效的邮箱。");
          return;
        }
      }
      if (data.updateInfo.nickname != "") {
        data.userInfo.nickname = data.updateInfo.nickname;
      }
      if (data.updateInfo.email != "") {
        data.userInfo.email = data.updateInfo.email;
      }
      if (data.fileList.length != 0) {
        try {
          data.updateInfo.avatar = "/static/avatars/" + data.fileList[0].name;
          console.log(data.updateInfo.avatar);
          data.userInfo.avatar = store.state.backendUrl + data.updateInfo.avatar;
          store.commit("setUserInfo", data.userInfo);
          data.uploadRef.submit();
        } catch (error) {
          console.log(error);
        }
      }

      if (data.updateInfo.birthday != "") {
        data.userInfo.birthday = data.updateInfo.birthday;
      }
      if (data.updateInfo.signature != "") {
        data.userInfo.signature = data.updateInfo.signature;
      }
      data.isMyInfoModalVisible = false;
      data.fileList = [];
      data.cnt = 0;
      data.updateInfo.uuid = data.userInfo.uuid;
      store.commit("setUserInfo", data.userInfo);
      try {
        const rsp = await axios.post(
          store.state.backendUrl + "/user/updateUserInfo",
          data.updateInfo
        );
        console.log(rsp);
        if (rsp.data.code == 200) {
          ElMessage.success(rsp.data.message);
        } else if (rsp.data.code == 400) {
          ElMessage.error(rsp.data.message);
        } else if (rsp.data.code == 500) {
          ElMessage.error(rsp.data.message);
        }
      } catch (error) {
        console.log(error);
      }
      router.go(0);
    };
    const quitMyInfoModal = () => {
      data.isMyInfoModalVisible = false;
      data.fileList = [];
      data.cnt = 0;
    };
    const handleUploadSuccess = () => {
      ElMessage.success("头像上传成功");
      data.fileList = [];
    };
    const beforeFileUpload = (file) => {
      console.log("上传前file====>", file);
      console.log(data.fileList);
      console.log(file);
      if (data.fileList.length > 1) {
        ElMessage.error("只能上传一张头像");
        return false;
      }
      const isLt50M = file.size / 1024 / 1024 < 50;
      if (!isLt50M) {
        ElMessage.error("上传头像图片大小不能超过 50MB!");
        return false;
      }
    };
    const getFileExtension = (filename) => {
      const parts = filename.split(".");
      return parts.length > 1 ? parts.pop() : "";
    };

    
    return {
      ...toRefs(data),
      router,
      showMyInfoModal,
      closeMyInfoModal,
      quitMyInfoModal,
      handleUploadSuccess,
      beforeFileUpload,
    };
  },
};
</script>

<style scoped>
.owner-info-window {
  width: 84%;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background-color: #FFFEF0; /* 浅米黄色 */
  border: 2px solid #8B7355; /* 像素边框 */
  padding: 16px;
  image-rendering: pixelated;
}

.owner-prefix {
  font-family: 'BoutiqueBitmap', monospace; /* 像素字体 */
  margin: 6px;
  color: #8B7355; /* 棕色 */
}

.owner-opt {
  margin: 6px;
  display: flex;
  flex-direction: row;
}

.edit-window {
  width: 10%;
  display: flex;
  flex-direction: column-reverse;
}

h3 {
  font-family: 'BoutiqueBitmap', monospace; /* 像素字体 */
  color: #8B7355; /* 棕色 */
}

.modal-quit-btn-container {
  height: 30%;
  width: 100%;
  display: flex;
  flex-direction: row-reverse;
}

.modal-quit-btn {
  background-color: #DAA520; /* 金黄 */
  color: #FFF;
  padding: 15px;
  border: 4px solid #8B7355; /* 像素边框 */
  cursor: pointer;
  position: fixed;
  justify-content: center;
  align-items: center;
  font-family: 'BoutiqueBitmap', monospace;
  image-rendering: pixelated;
  box-shadow: 4px 4px 0px #8B7355;
  border-radius: 4px;
}

.modal-quit-btn:hover {
  background-color: #B8860B;
  transform: translate(2px, 2px);
  box-shadow: 2px 2px 0px #8B7355;
}

.modal-header {
  height: 20%;
  width: 100%;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  background-color: #FFFEF0; /* 浅米黄色 */
  border: 4px solid #8B7355; /* 像素边框 */
  border-radius: 4px;
  image-rendering: pixelated;
}

.modal-body {
  height: 100%;
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background-color: #FFF; /* 白色 */
  border: 4px solid #8B7355; /* 像素边框 */
  border-radius: 4px;
  margin-top: 8px;
  image-rendering: pixelated;
}

.modal-footer {
  height: 20%;
  width: 100%;
  display: flex;
  justify-content: center;
  align-items: center;
  background-color: #FFFEF0; /* 浅米黄色 */
  border: 4px solid #8B7355; /* 像素边框 */
  border-radius: 4px;
  margin-top: 8px;
  image-rendering: pixelated;
}

.modal-header-title {
  height: 70%;
  width: 100%;
  display: flex;
  justify-content: center;
  align-items: center;
  font-family: 'Courier New', monospace; /* 像素字体 */
  color: #8B7355; /* 棕色 */
}

h2 {
  margin-bottom: 20px;
  font-family: 'Courier New', monospace; /* 像素字体 */
  color: #8B7355; /* 棕色 */
}

.el-menu {
  background-color: #FFFEF0; /* 浅米黄色 */
  width: 101%;
  border: 4px solid #8B7355; /* 像素边框 */
  border-radius: 4px;
  image-rendering: pixelated;
}

.el-menu-item {
  background-color: #FFF; /* 白色 */
  height: 45px;
  border-bottom: 4px solid #8B7355; /* 像素底部边框 */
  font-family: 'Courier New', monospace;
  image-rendering: pixelated;
}

.el-menu-item:last-child {
  border-bottom: none; /* 最后一个项目无边框 */
}
</style>