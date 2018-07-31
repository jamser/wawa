$(function(){

	/*$(".equipment-delete").click(function () {
		$(this).parent().parent().parent().remove();
	})*/


	// $(".equipment-alter").click(function () {
	// 	if (this.innerHTML == "点击修改") {
	// 		//$(this).parent().find(".equipment-off").show();
	// 		$(this).text('完成修改');
	// 		$(this).parent().find("li input").removeAttr("readonly");
	// 		$(this).parent().find("li textarea").removeAttr("readonly");
	// 		$(this).parent().find(".munber-played").attr("readonly","readonly");
	// 	}
	// 	else{
	// 		//$(this).parent().find(".equipment-off").hide();
	// 		$(this).text('点击修改');
	// 		$(this).parent().find("li input").attr("readonly","readonly");
	// 		$(this).parent().find("li textarea").attr("readonly","readonly");
	// 	}
	// })


    //编辑按钮事件绑定
    $(".edit").click(function () {
    	editLobby(this);
    });
    $("#update").click();
   	$("#update").on("change", function () {
    	loadImg();
    });

    $("#update_describe").click();
   	$("#update_describe").on("change", function () {
    	loadImg_describe();
    });

    $(".equitment_edit").click(function () {
    	editLobby_equitment(this);
    })

    $(".change-img").click(function () {
    	editLobby_active(this);
    })
})
	//编辑大厅
//isvisible和name从table的data-*取出
function editLobby(obj) {
	$table = $(obj).prevAll('table');
	var url = $table.find('[name="picture"]').attr("src");
	var data = {
		'id': $table.find('[name="id"]').text().trim(),
		'name': $table.find('[name="name"]').text().trim(),
		'type': $table.find('[name="type"]').text().trim(),
		'amount': $table.find('[name="amount"]').text().trim(),
		'obtained': $table.find('[name="obtained"]').text().trim(),
		'date_obtained': $table.find('[name="date_obtained"]').text().trim(),
		'introduce': $table.find('[name="introduce"]').text().trim(),	
	};
	initEditModal(data,url);
	 
}

//初始化并显示编辑模态框
function initEditModal(data,url) {
	$modal = $('#editModal');
	$.each(data, function (key, value) {
		$modal.find('[name="'+ key +'"]').val(value);
		if (key == "id") return true;
		$modal.find('[name="'+ key +'"]').val(value);
	});
	$modal.find("[name='picture']").attr("src",url);
	$modal.modal('show');
}

function loadImg(){  
    //获取文件  
    var file = $("#update")[0].files[0];  
  
    //创建读取文件的对象  
    var reader = new FileReader();  
  
    //创建文件读取相关的变量  
    var imgFile;  
  
    //为文件读取成功设置事件  
    reader.onload=function(e) {  
        imgFile = e.target.result;  
        console.log(imgFile);  
        $("#picture").attr('src', imgFile);  
    };  
  
    //正式读取文件  
    reader.readAsDataURL(file);  
}  
function loadImg_describe(){  
    var file = $("#update_describe")[0].files[0];  
    var reader = new FileReader();
    var imgFile;
    reader.onload=function(e) {  
        imgFile = e.target.result;  
        console.log(imgFile);  
        $("#picture_describe").attr('src', imgFile);  
    };  
    reader.readAsDataURL(file);  
}  


function editLobby_equitment(obj) {
	$table = $(obj).prevAll('table');
	var url = $table.find('[name="picture"]').attr("src");
	var	describe = $table.find('[name="describe"]').attr("src");
	var data = {
		'id': $table.find('[name="id"]').text().trim(),
		'name': $table.find('[name="name"]').text().trim(),
		'price': $table.find('[name="price"]').text().trim(),
		'type': $table.find('[name="type"]').text().trim(),
		'remainder': $table.find('[name="remainder"]').text().trim(),
		'power': $table.find('[name="power"]').text().trim(),
		'exchange': $table.find('[name="exchange"]').text().trim(),
	};
	initEditModal_equitment(data,url,describe);
	 
}

//初始化并显示编辑模态框
function initEditModal_equitment(data,url,describe) {
	$modal = $('#editModal_equitment');
	$.each(data, function (key, value) {
		$modal.find('[name="'+ key +'"]').val(value);
	});
	$modal.find("[name='picture']").attr("src",url);
	$modal.find("[name='describe']").attr("src",describe);
	$modal.modal('show');
}



function editLobby_active(obj) {
	$table = $(obj).prevAll('table');
	var url = $table.find('[name="img"]').attr("src");
	var data = {
		'id': $table.find('[name="id"]').text().trim(),
	}
	initEditModal_active(url,data);
	 
}

//初始化并显示编辑模态框
function initEditModal_active(url,data) {
	$modal = $('#editModal_active');
	$modal.find("[name='img']").attr("src",url);
	$.each(data, function (key, value) {
		$modal.find('[name="id"]').val(value);
	});
	$modal.modal('show');
}